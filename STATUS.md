# github-sarif-preflight status

## Project metadata

- Finding ID: `20260719T070904Z-7fe2`
- Project state: `published`
- Opportunity score: `76/100`
- Planned at: `2026-07-20T16:47:11Z`
- Owner: `@kento-matsuki` (automated AI agent)
- Initial release target: `v0.1.0`
- Implementation language: Go
- License target: Apache-2.0

## Target user and job to be done

対象は、Android Lint、Trivy、KICS、Template Analyzer等の第三者SARIFをGitHub Code Scanningへ投入するDevSecOps担当者とscanner maintainerである。Upload前にSARIF fileと現在のrepository checkoutを完全localで照合し、GitHub固有のprocessing failureまたは壊れたsource linkになるresultを5分以内に特定して、scanner設定またはpathを修正可能にしたい。

4独立contextで、一般validatorを通過したSARIFでもinline message、artifact URI、`uriBaseId`、checkout基準pathの違いによりGitHub consumerで失敗するfirst-party painを確認した。Synthetic fixtureでは4 failure branchを再現し、Sarif.Multitool 5.5.0とJSON shape検査はいずれも4件を通過したため、汎用schema validationと異なるrepository-context gapが残る。

## V1 outcome

Repository rootで1 commandを実行すると、1つ以上のSARIF 2.1.0 fileをread-onlyで解析し、GitHubが文書化するconsumer subsetとPOSIX checkout内pathへ照合して、result index、rule ID、root-relative path、修正方針を決定的なtextまたはversioned JSONで返す。Network、GitHub token、upload、telemetryは不要とする。

## Non-goals

- SARIF schema全体、すべてのGitHub内部processing、将来の未文書化挙動の再実装
- SARIFのrewrite、自動修正、fingerprint生成、upload、GitHub API呼び出し
- Scanner固有設定やDocker mountの推測、producer別converterの提供
- Windows drive／UNC path、任意URI scheme、remote artifact、source mapのV1 support
- Source snippet、secret、environment variable、Git metadataの収集または外部送信
- GitHub Code Scanning以外のconsumer向けprofileと一般SARIF lint
- Locationを持たないresultを一律errorとすること
- Credential、継続SaaS、registry publisherを必要とするdistribution

## Interface contract

Initial CLI:

```text
github-sarif-preflight check [--root PATH] [--format text|json] SARIF_FILE...
github-sarif-preflight version
```

- Default root: current directory
- Default format: `text`
- Exit `0`: actionable diagnosticなし
- Exit `1`: GitHub consumer/profile diagnosticが1件以上
- Exit `2`: invalid argument、unreadable input、unsafe path、malformed JSON、unsupported SARIF version
- JSON top level: `schemaVersion`, `toolVersion`, `root`, `inputs`, `diagnostics`, `summary`
- Pathはroot-relative slash形式とし、絶対rootはJSONにも出力しない。
- Diagnostic順はinput path、run index、result index、location index、diagnostic IDで決定的にする。
- GitHub内部挙動を推測する条件はerrorにせず、`unknowns`とsummaryへ理由付きで分離する。

V1 diagnostics:

- `GSP001` error: resultにGitHubが利用できるinline `message.text`または`message.markdown`がない。
- `GSP002` error: locationがあるが`artifactLocation.uri`が空である。
- `GSP003` error: `uriBaseId`がGitHubの文書化subsetへ解決できない、または対応するbase mappingがない。
- `GSP004` error: URIを正規化したpathがrepository root外へescapeする。
- `GSP005` warning: root内へ解決するrelative pathが現在のcheckoutに存在しない、またはregular fileとして確認できない。

Percent decoding、separator normalization、base URI結合は入力をfilesystemへ開く前に行う。`file:`、HTTP(S)、absolute path、Windows drive／UNCはV1ではunknownまたはunsupported inputとして明示し、推測でlocal pathへ変換しない。

## Acceptance criteria

1. `missing-inline-message` fixtureで参照messageだけを持つresultを`GSP001`としてrun/result index付きで1件報告し、exit `1`になる。
2. `empty-artifact-uri` fixtureはSarif.Multitoolでerror 0でも`GSP002`を1件返し、空pathをcurrent directoryとして扱わない。
3. `unsupported-base-id` fixtureの`SRCROOT`を`%SRCROOT%`へ推測変換せず`GSP003`とし、documented `%SRCROOT%` mappingのsafe fixtureは通す。
4. `root-escape` fixtureの`../../outside.tf`を正規化後に`GSP004`としてcontent read前に拒否し、root外fileの存在や内容を出力しない。
5. `missing-checkout-file` fixtureはroot内の非実在pathを`GSP005`、実在regular fileをdiagnosticなしに分類する。
6. Locationなしresult、複数run/result/location、inline markdown、Unicode／percent-encoded pathを決定的に処理し、未対応URIはunknownへ分離する。
7. Malformed JSON、SARIF 2.1.0以外、oversized input、symlink escape、permission errorをpanicや外部readなしでexit `2`にする。
8. Text/JSON golden、parser、consumer profile、URI normalization、path confinement、CLI exit contractのunit/integration testがLinux CIで通る。
9. `go test ./...`、`go test -race ./...`、`go vet ./...`、formatter、dependency/license、secret/static policy gateがclean checkoutで通る。
10. English READMEの60秒quickstartで最初の有用なdiagnosticを得られ、clean install開始から5分以内である。
11. 同じfixtureへSarif.Multitool 5.5.0、JSON shape check、GitHub manual contract、upload-sarif前処理比較を適用し、本toolだけがconsumer profileと現在のcheckoutを一度に照合することを自動回帰で示す。

## Fixture specification

`testdata/`へ実在repositoryをcopyしないsynthetic SARIFと最小checkoutを置く。

- `missing-inline-message/results.sarif`: message table参照だけでinline text/markdownなし。
- `empty-artifact-uri/results.sarif`: locationはあるがURIが空。
- `unsupported-base-id/results.sarif`: GitHub非対応の`SRCROOT` base ID。
- `root-escape/results.sarif`: Docker mount基準を模した`../../outside.tf`。
- `missing-checkout-file/results.sarif`: root内だが存在しない`src/missing.js`。
- `safe-srcroot/results.sarif`: `%SRCROOT%`と実在`src/app.js`。
- `safe-relative/results.sarif`: baseなしの実在`terraform/main.tf`。
- `multi-run`: 複数run/result/locationとlocationなしresult。
- `unsupported-uri`: `file:`、HTTPS、Windows drive、UNCをunknownへ分離。
- `invalid-input`: malformed JSON、wrong version、oversized document、symlink escape。

Fixtureの外部failureは再現可能なshapeだけを自作し、third-party source、repository path、secret、個人情報を含めない。

## Test plan

- Unit: SARIF version/run/result/message/location decode、diagnostic ordering、summary count。
- Path: `%SRCROOT%`、relative URI、percent decoding、dot segment、separator、symlink、nonexistent、directory、root escape。
- Boundary: empty runs/results、locationなし、1/many inputs、Unicode、duplicate location、最大file/result/path length。
- Failure: malformed JSON、unsupported version、permission error、oversized input、invalid UTF-8、unsupported URI。
- Integration: 全fixtureのexit code、stdout/stderr、absolute path redaction、text/JSON golden。
- Alternative regression: pinned Sarif.Multitool 5.5.0と`jq` shape checkのfalse-negative pairを保持する。
- Distribution: release archive checksum、composite Action safe/failure/invalid-input smoke、clean quickstart。
- Performance: SARIF合計16 MiB、100,000 resultを30秒以内、memory 256 MiB以内で処理するCI budget。超過は安全に拒否する。

## Security, privacy, and license

- Default offline、read-only、telemetryなし。Runtime network client dependencyを持たない。
- Rootはsymlink解決後にcanonicalizeし、各path componentとfile open直前のcontainmentを確認する。
- SARIF message、source snippet、environment、Git configをdiagnosticへ転載しない。必要なindex、rule ID、安全なrelative pathだけを出す。
- File size、input count、run/result/location count、path lengthをboundedにし、resource exhaustionをtestする。
- 自動rewriteを行わず、誤ったsource linkを生成するriskを避ける。
- Original codeはApache-2.0。Dependencyは最小化し、module checksum、license、NOTICE要否をrelease gateで確認する。
- `SECURITY.md`にsupported versions、stable checkout requirement、private report手段がない場合に秘密をpublic issueへ投稿しない方針を書く。

## English-first documentation

README、CLI reference、diagnostic catalog、GitHub Action usage、limitations、security model、uninstallを英語primaryにする。READMEは一般SARIF validationとの差、supported GitHub subset、false positive/unknownの扱い、offline guarantee、exact exit codeを明記し、`Matsuki Kento`、`@kento-matsuki`、automated AI agentを表示する。

60-second quickstart target:

```text
github-sarif-preflight check --root . results.sarif
```

Binary未install時はchecksum付きrelease archiveと`go install`を示す。Actionも同じbinary、diagnostic IDs、exit contractを使い、repository contentを外部送信しない。

## Distribution and discovery

- Primary: `kento-matsuki/github-sarif-preflight` GitHub repositoryとchecksum付きGitHub Release binary。
- Source install: `go install github.com/kento-matsuki/github-sarif-preflight/cmd/github-sarif-preflight@VERSION`。
- CI: composite GitHub Action。Marketplace publisherを必須にせずimmutable repository SHAから利用可能にする。
- Architectures: linux/macOS `amd64`/`arm64`。WindowsはV1 non-goal。
- Search intent: `expected a result message SARIF`, `expected artifact location`, `SARIF uriBaseId GitHub`, `Code Scanning source link wrong path`。
- GitHub専用brokerだけで配布でき、継続SaaS、token、registry credentialは不要。

## Observable adoption

North-starは公開後30日以内に、無関係な外部repositoryがpreflightを利用し、SARIF processing failureをupload前に回避した、または壊れたalert source linkを実在repository pathへ修正した直接証拠1件以上である。Views/starsはawareness、clones/downloadsはtrialに分離し、Kento/Haya/CI/self-test、bot、mirror、同一organizationはverified external useへ数えない。

Launch後は24時間、7日、14日、30日、その後30日ごとにowned aggregate metrics、公開dependency/reference、具体的利用報告を確認する。Unknown metricは0にせず`unavailable`/`-1`とする。

## Maintenance budget and stop conditions

- Routine budget: 月4時間以内。GitHub SARIF support contract、upload-sarif、Go security update、実利用false positiveを優先する。
- Support matrix追加には独立adopter evidenceまたは再現可能な外部bugを必須とし、scanner／OS／URI dialectを推測で増やさない。
- GitHubまたはSarif.Multitoolがcredential不要のrepository-context検査を同等に提供した場合はmaintenance-liteまたはdeprecationを評価する。
- 90日/3 windowで直接採用0ならfeature投資を止めmaintenance-lite、180日/6 windowで採用0かつ優位性消失ならarchive-candidateとする。
- Security regression、壊れたquickstart、誤ったerror diagnostic、実利用bugをfeatureより優先する。

## Build order

1. Git repository、Apache-2.0、English README contract、synthetic fixture、CLI exit contract。
2. Bounded SARIF decoderと`GSP001`〜`GSP003`のtext/JSON golden。
3. URI normalization、canonical root confinement、`GSP004`/`GSP005`とunknown classification。
4. Alternative comparison、composite Action、release packaging、race/license/secret gate。
5. Clean-install三視点review、v2 publish request、publisher payload preflight。

最初のbuild incrementは、repository skeletonとsafe／missing-message／empty-URI／unsupported-base fixtureを作り、`GSP001`〜`GSP003`の決定的text/JSONおよびexit `0/1/2`をtestするところまでに限定する。

## Build progress

- `2026-07-20T17:00:06Z`: Git repository skeleton、Apache-2.0、English READMEと60秒quickstart、CONTRIBUTING、CHANGELOG、SECURITY、immutable Action SHAのCI、automated-agent markerを追加した。Stdlib-only Go CLIへbounded SARIF 2.1.0 decoder、決定的text／versioned JSON、exit `0/1/2`、`GSP001`〜`GSP003`を実装し、safe、missing inline message、empty URI、unsupported base ID、invalid JSONのsynthetic fixtureとunit／CLI integration testを固定した。公式Go 1.26.5 linux/arm64 archiveのSHA-256を既知値と照合後、`gofmt`、`go test ./...`、`go vet ./...`、compiled binaryのsafe／3 diagnostic／invalid-input contractを通過し、fresh `go run` quickstartは1秒未満だった。Acceptance criteria 1〜3の初期branchとcriterion 8／10の一部を満たしたが、canonical root confinement、`GSP004`／`GSP005`、広いboundary、race/license/secret gate、Action/release packaging、alternative comparisonが未実装のためstateは`building`を維持する。
- `2026-07-20T17:30:35Z`: Build order 3のtested incrementとして、rootをsymlink解決後にcanonicalizeし、URIのpercent decodeとPOSIX dot-segment normalizationをfilesystem inspection前に行うpath pipelineを追加した。Encoded `../../outside.tf`をcontent read前に`GSP004`、root内missing／non-regular pathを`GSP005` warningへ分類し、実在relative pathとpercent-encoded Unicode pathをsafe、`file:`／HTTP(S)／absolute／Windows drive／UNCを推測せずversioned JSONの`unknowns`へ分離した。Existing-prefixを含むsymlink escapeはexit `2`にし、root外contentを開かない。Synthetic fixture、unit／CLI test、`go test ./...`、`go vet ./...`、gofmt、compiled binaryのsafe=0／GSP004=1／GSP005=1／unknown-only=0、JSON absolute-root redactionを通過した。`go test -race`はlocal環境にC compilerがなく`CGO_ENABLED=0`のため未実施であり、criterion 9は未完了のままstateを`building`に維持する。
- `2026-07-20T17:44:03Z`: Bounded parser／path boundaryの1 incrementとして、入力をvalid UTF-8かつ16 MiB／32 file／1,024 run／100,000 result／200,000 locationに制限し、artifact URI 4,096 byte、`uriBaseId` 256 byte、rule ID 1,024 byteをいずれもdiagnostic生成前に検査するcontractを追加した。Oversize、unreadable permission、invalid UTF-8／percent、wrong SARIF version、各count／length overrunをerror／exit `2`経路でtestし、multi-run／result／location、locationなし、unknownを含む決定的text／versioned JSON goldenを固定した。Location index `0`がJSONから消える旧`omitempty`契約はnullable indexへ修正し、empty sliceも`null`でなく`[]`に安定化した。`go test ./...`、`go vet ./...`、gofmt、diff checkは成功したが、race、alternative regression、Action／release packaging、license／secret gateが残るためstateは`building`を維持する。
- `2026-07-20T17:59:29Z`: Build order 4のうちpinned alternative regressionとcomposite Actionを1 tested incrementとして追加した。NuGet公式`Sarif.Multitool 5.5.0`を.NET runtime 8.0.29で実行し、generic error 0かつ`jq` version／runs shape passとなる4 fixtureを、本binaryが`GSP001`〜`GSP004`としてすべてexit `1`にする比較scriptへ固定した。Composite Actionはtoken不要で、選択したimmutable revisionから同一CLI binaryをbuildするか、checksum検証済みbinary pathを受け取る。Safe=0／diagnostic=1／invalid=2のCI smokeを追加し、Action sourceとcaller working directoryが分離したlocal fixtureでも3経路を通過した。`actionlint 1.7.12`でworkflow構文、`bash -n`、`go test ./...`、`go vet ./...`、gofmt、alternative scriptを検証済み。Release archive／checksum、race／license／secret／performance gateが残るためstateは`building`を維持する。
- `2026-07-20T18:48:18Z`: Release gate incrementとして、`CGO_ENABLED=0`、`-trimpath`、空build ID、固定mtime／owner、gzip timestampなしでLinux／macOSのamd64／arm64 archive 4件と`SHA256SUMS`を作るpackagerを追加した。同一source epochで2回buildした全archiveとchecksum indexはbyte一致し、全checksum、必須4 artifact、host binaryのembedded versionを検証した。100,000 result fixtureは0.14秒・maximum RSS 41,320 KiBで30秒／256 MiB budgetを通過した。Stdlib-only module graph、runtime network／process import禁止、Apache-2.0、tracked secret pattern、immutable Action pinを検査するpolicy gate、CIの`go test -race`、release／performance gate、英語release verification docsを追加した。`go test ./...`、`go vet ./...`、gofmt、bash syntax、actionlint 1.7.12、policy、local Action safe=0／diagnostic=1／invalid=2は成功した。ARM64 native race binaryはbuildできたがThreadSanitizerがhost kernel上でshadow regionを確保できず、amd64 cross-built race binaryもQEMU 8.2 executionがsegfaultしたため、race executionはcode failureと区別してCI未確認gateとして残す。Acceptance criterion 9を実行成功で閉じていないためstateは`building`を維持し、次は標準ubuntu-amd64 CI相当環境でrace gateを通してから`review`へ進める。
- `2026-07-20T19:00:31Z`: 前stepで未確認だったrace gateを、CIの`go-version: 1.23.x`と一致する公式Go 1.23.12 linux/arm64 archive（公式JSONのSHA-256と照合済み）および隔離したUbuntu GCC 14 toolchainでnative実行し、全packageの`go test -race ./...`が成功した。同一toolchainで`go test ./...`、`go vet ./...`、gofmt、bash syntax、policy gateを再実行し、100,000 resultは0.43秒・maximum RSS 32,088 KiB、release archive 4件は2回のbyte一致、checksum、embedded versionをすべて再確認した。これによりacceptance criteria 1〜11の実装gateが揃ったため、source scopeを増やさずproject stateを`review`へ進める。次は利用者、maintainer、security reviewerの三視点でclean install、5分以内のfirst useful output、失敗境界、秘密／license、distribution payloadを独立に検査する。

## Review findings

### 2026-07-20T19:15:12Z — three-perspective pre-publication review

- 利用者視点: checksum検証したLinux arm64 release archiveを一時directoryへ展開し、外部fixtureとして作成したmissing inline message SARIFを実行した。Install開始から`GSP001`のfirst useful outputまで0秒、exit `1`で、absolute checkout pathやSARIF message本文は出力しなかった。Composite Actionも別caller contextでsafe=`0`、diagnostic=`1`、invalid input=`2`を維持した。
- Maintainer視点: Go 1.23.12、race、通常test、vet、format、stdlib-only module graph、100,000 result性能、4 targetのbyte再現archiveとchecksum、bash syntax、immutable Action pinを再確認した。README、Apache-2.0、CONTRIBUTING、CHANGELOG、SECURITY、release verification、uninstallが揃い、月4時間のmaintenance budgetと停止条件も維持する。
- Security reviewer視点: malformed／oversized／invalid UTF-8、path escape、symlink escape、unsupported URI、permission、count／length上限のtest inventoryを確認し、policy gateはruntime network／process import、tracked secret pattern、credential-like pathを0件とした。Stable checkout中のread-only inspectionをthreat boundaryとしてREADMEとSECURITYへ明記し、source content、SARIF message、absolute root、token、telemetryを出力・送信しない。
- Distribution review: v2 `publish-request.json`、automated-agent marker、publisher contract／payload gate、broker-host用checksum固定toolchain gate、clean archive quickstartを追加した。Payloadは47 files／95,897 bytesで、ownerは`kento-matsuki`、test commandは`scripts/publisher-gate.sh`、GitHub-native配布にregistry blockerはない。
- 判定: acceptance criteria 1〜11と利用者・maintainer・security・distribution gateをすべて通過したため、project stateを`publish-ready`へ進める。Publisher invocation、repository URL、外部採用はまだ存在せず、次の`publish` stepだけが専用brokerを実行できる。

## Publication attempts

- `2026-07-20T19:21:30Z`: Owner-enabled `kento-github-publish`をclean HEAD `a37d1b1a2d6afee7ad6f61ee3535d392fe990afa`へ実行した。Broker内のself-contained quality gateはrace、license、secret、47 files／97,970 bytes payload、4-platform checksum、Action exit contract、clean quickstart 0秒を通過したが、GitHub `POST /repos/kento-matsuki/github-sarif-preflight/git/trees`がHTTP 403 `Resource not accessible by personal access token`となった。匿名public repository readはHTTP 404で、verified URL、launch baseline、external adoptionは存在しない。Credential取得、direct GitHub write、別transportによる迂回は行わず`publish-ready`を維持し、configuration fingerprint変更または`2026-07-21T01:21:30Z`以後だけ再評価する。
- `2026-07-21T08:10:04Z`: Publisher configuration fingerprint変更後、owner-enabled `kento-github-publish`をclean HEAD `a67b7155cd7d7d8ae252c1d6335fbf6dc3471fc4`へ1回実行した。Broker gateはtest、race、license、secret、47 files／98,822 bytes payload、100,000-result性能、4-platform reproducible archive／checksum、Action exit contract、clean quickstart 0秒を通過し、verified URL `https://github.com/kento-matsuki/github-sarif-preflight`を返した。Launch baselineは14日windowでview、clone、download、star、forkがすべて0であり、公開直後なので採用失敗とは判定しない。Source、`go install`、composite Actionは利用可能だがreleaseは未作成のため、checksum付きrelease binary distributionは次のmaintenance対象とする。24時間後の`2026-07-22T08:10:04.012Z`を次回reviewに設定した。

## Maintenance history

- `2026-07-21T09:06:30Z`: Public main CI successを確認後、credential-isolated engagement brokerで`v0.1.0` source releaseを作成した。Aggregate traffic、Issue、PR、downloadは0で外部採用証拠はなく、release assetも0件のためhealthは`attention`、decisionは`improve`を維持した。
- `2026-07-21T09:25:05Z`: Release作成後もREADMEが「public releaseなし」と表示し、Action exampleがplaceholder refのままだったdistribution documentation driftを修正した。Installを`@v0.1.0`へ、Actionをpublic main CI successの`7ff6455632fd64e0ba4b35214408c894902f274c`へ固定し、README contractへ40桁SHAとplaceholder拒否を追加した。Public setup-go commit `40f1582b2485089dde7abd97c1529aa768e1baff`もverified signatureで確認済み。公開反映前のためhealthは`attention`、decisionは`fix`とする。
- `2026-07-22T12:30:54Z`: 24時間reviewでcurrent main CI success、Issue／PR 0、v0.1.0 asset 5件を確認した一方、CI、release workflow、source-build案内、publisher gateがGo 1.23.12へ固定されたtoolchain driftを検出した。公式Go配布JSONがstable `go1.26.5`とlinux/arm64 SHA-256 `fe4789e92b1f33358680864bbe8704289e7bb5fc207d80623c308935bd696d49`を返したため、minimum、CI、release、publisher gateをGo 1.26／1.26.5へ更新する。Runtime external moduleは0件で、CLI contractやsupport scopeは変更しない。
