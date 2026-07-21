package preflight

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	maxInputBytes       = 16 << 20
	maxRuns             = 1_024
	maxResults          = 100_000
	maxLocations        = 200_000
	maxArtifactURIBytes = 4_096
	maxURIBaseIDBytes   = 256
	maxRuleIDBytes      = 1_024
)

type sarifLog struct {
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Results []sarifResult `json:"results"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	Message   sarifMessage    `json:"message"`
	Locations []sarifLocation `json:"locations"`
}

type sarifMessage struct {
	Text     string `json:"text"`
	Markdown string `json:"markdown"`
}

type sarifLocation struct {
	PhysicalLocation *physicalLocation `json:"physicalLocation"`
}

type physicalLocation struct {
	ArtifactLocation *artifactLocation `json:"artifactLocation"`
}

type artifactLocation struct {
	URI       *string `json:"uri"`
	URIBaseID string  `json:"uriBaseId"`
}

// Diagnostic is intentionally content-safe: it never contains SARIF messages or source text.
type Diagnostic struct {
	ID          string `json:"id"`
	Severity    string `json:"severity"`
	Input       string `json:"input"`
	Run         int    `json:"run"`
	Result      int    `json:"result"`
	Location    *int   `json:"location,omitempty"`
	RuleID      string `json:"ruleId,omitempty"`
	Path        string `json:"path,omitempty"`
	Message     string `json:"message"`
	Remediation string `json:"remediation"`
}

type Summary struct {
	Inputs      int `json:"inputs"`
	Runs        int `json:"runs"`
	Results     int `json:"results"`
	Diagnostics int `json:"diagnostics"`
	Unknowns    int `json:"unknowns"`
}

// Unknown records an intentionally unsupported URI without guessing that it is local.
type Unknown struct {
	Input    string `json:"input"`
	Run      int    `json:"run"`
	Result   int    `json:"result"`
	Location int    `json:"location"`
	RuleID   string `json:"ruleId,omitempty"`
	Reason   string `json:"reason"`
}

type Report struct {
	SchemaVersion int          `json:"schemaVersion"`
	ToolVersion   string       `json:"toolVersion"`
	Root          string       `json:"root"`
	Inputs        []string     `json:"inputs"`
	Diagnostics   []Diagnostic `json:"diagnostics"`
	Unknowns      []Unknown    `json:"unknowns"`
	Summary       Summary      `json:"summary"`
}

func Analyze(root, version string, inputs []string) (Report, error) {
	report := Report{
		SchemaVersion: 1,
		ToolVersion:   version,
		Root:          ".",
		Inputs:        append([]string(nil), inputs...),
		Diagnostics:   []Diagnostic{},
		Unknowns:      []Unknown{},
	}
	if strings.TrimSpace(root) == "" {
		return report, errors.New("root must not be empty")
	}
	if len(inputs) == 0 {
		return report, errors.New("at least one SARIF file is required")
	}
	if len(inputs) > 32 {
		return report, errors.New("at most 32 SARIF files are supported")
	}
	canonicalRoot, err := canonicalizeRoot(root)
	if err != nil {
		return report, err
	}

	locationsSeen := 0
	for _, input := range inputs {
		log, err := readLog(input)
		if err != nil {
			return report, fmt.Errorf("%s: %w", input, err)
		}
		report.Summary.Inputs++
		if report.Summary.Runs+len(log.Runs) > maxRuns {
			return report, fmt.Errorf("at most %d SARIF runs are supported per invocation", maxRuns)
		}
		report.Summary.Runs += len(log.Runs)
		for runIndex, run := range log.Runs {
			if report.Summary.Results+len(run.Results) > maxResults {
				return report, fmt.Errorf("at most %d SARIF results are supported per invocation", maxResults)
			}
			report.Summary.Results += len(run.Results)
			for resultIndex, result := range run.Results {
				if len(result.RuleID) > maxRuleIDBytes {
					return report, fmt.Errorf("%s: run[%d].result[%d]: rule ID exceeds %d bytes", input, runIndex, resultIndex, maxRuleIDBytes)
				}
				if locationsSeen+len(result.Locations) > maxLocations {
					return report, fmt.Errorf("at most %d SARIF locations are supported per invocation", maxLocations)
				}
				locationsSeen += len(result.Locations)
				if strings.TrimSpace(result.Message.Text) == "" && strings.TrimSpace(result.Message.Markdown) == "" {
					report.Diagnostics = append(report.Diagnostics, Diagnostic{
						ID: "GSP001", Severity: "error", Input: input, Run: runIndex, Result: resultIndex, RuleID: result.RuleID,
						Message:     "result has no inline message.text or message.markdown",
						Remediation: "add an inline result message before uploading to GitHub Code Scanning",
					})
				}
				for locationIndex, location := range result.Locations {
					if location.PhysicalLocation == nil || location.PhysicalLocation.ArtifactLocation == nil {
						continue
					}
					artifact := location.PhysicalLocation.ArtifactLocation
					path := ""
					if artifact.URI != nil {
						path = strings.TrimSpace(*artifact.URI)
					}
					if path == "" {
						report.Diagnostics = append(report.Diagnostics, Diagnostic{
							ID: "GSP002", Severity: "error", Input: input, Run: runIndex, Result: resultIndex, Location: intPointer(locationIndex), RuleID: result.RuleID,
							Message:     "physical location has an empty artifactLocation.uri",
							Remediation: "set artifactLocation.uri to a non-empty repository-relative path",
						})
					}
					if len(path) > maxArtifactURIBytes {
						return report, fmt.Errorf("%s: run[%d].result[%d].location[%d]: artifact URI exceeds %d bytes", input, runIndex, resultIndex, locationIndex, maxArtifactURIBytes)
					}
					if len(artifact.URIBaseID) > maxURIBaseIDBytes {
						return report, fmt.Errorf("%s: run[%d].result[%d].location[%d]: uriBaseId exceeds %d bytes", input, runIndex, resultIndex, locationIndex, maxURIBaseIDBytes)
					}
					if artifact.URIBaseID != "" && artifact.URIBaseID != "%SRCROOT%" {
						report.Diagnostics = append(report.Diagnostics, Diagnostic{
							ID: "GSP003", Severity: "error", Input: input, Run: runIndex, Result: resultIndex, Location: intPointer(locationIndex), RuleID: result.RuleID, Path: path,
							Message:     fmt.Sprintf("unsupported GitHub Code Scanning uriBaseId %q", artifact.URIBaseID),
							Remediation: "use the documented %SRCROOT% base ID or a repository-relative URI",
						})
						continue
					}
					if path == "" {
						continue
					}
					normalized, state, err := inspectArtifact(canonicalRoot, path)
					if err != nil {
						return report, fmt.Errorf("%s: run[%d].result[%d].location[%d]: %w", input, runIndex, resultIndex, locationIndex, err)
					}
					switch state {
					case artifactUnknown:
						report.Unknowns = append(report.Unknowns, Unknown{
							Input: input, Run: runIndex, Result: resultIndex, Location: locationIndex, RuleID: result.RuleID,
							Reason: normalized,
						})
					case artifactEscapes:
						report.Diagnostics = append(report.Diagnostics, Diagnostic{
							ID: "GSP004", Severity: "error", Input: input, Run: runIndex, Result: resultIndex, Location: intPointer(locationIndex), RuleID: result.RuleID, Path: normalized,
							Message:     "artifact path escapes the repository root after normalization",
							Remediation: "emit a repository-relative artifact URI that remains inside the checkout",
						})
					case artifactMissing:
						report.Diagnostics = append(report.Diagnostics, Diagnostic{
							ID: "GSP005", Severity: "warning", Input: input, Run: runIndex, Result: resultIndex, Location: intPointer(locationIndex), RuleID: result.RuleID, Path: normalized,
							Message:     "artifact path is missing from the checkout or is not a regular file",
							Remediation: "make the scanner URI relative to the checked-out repository and point it to a regular file",
						})
					}
				}
			}
		}
	}

	sort.SliceStable(report.Diagnostics, func(i, j int) bool {
		a, b := report.Diagnostics[i], report.Diagnostics[j]
		if a.Input != b.Input {
			return a.Input < b.Input
		}
		if a.Run != b.Run {
			return a.Run < b.Run
		}
		if a.Result != b.Result {
			return a.Result < b.Result
		}
		if diagnosticLocation(a) != diagnosticLocation(b) {
			return diagnosticLocation(a) < diagnosticLocation(b)
		}
		return a.ID < b.ID
	})
	sort.SliceStable(report.Unknowns, func(i, j int) bool {
		a, b := report.Unknowns[i], report.Unknowns[j]
		if a.Input != b.Input {
			return a.Input < b.Input
		}
		if a.Run != b.Run {
			return a.Run < b.Run
		}
		if a.Result != b.Result {
			return a.Result < b.Result
		}
		return a.Location < b.Location
	})
	report.Summary.Diagnostics = len(report.Diagnostics)
	report.Summary.Unknowns = len(report.Unknowns)
	return report, nil
}

type artifactState int

const (
	artifactSafe artifactState = iota
	artifactMissing
	artifactEscapes
	artifactUnknown
)

func canonicalizeRoot(root string) (string, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve root: %w", err)
	}
	canonical, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", fmt.Errorf("resolve root: %w", err)
	}
	info, err := os.Stat(canonical)
	if err != nil {
		return "", fmt.Errorf("inspect root: %w", err)
	}
	if !info.IsDir() {
		return "", errors.New("root must be a directory")
	}
	return filepath.Clean(canonical), nil
}

// inspectArtifact performs URI and lexical confinement checks before touching the
// candidate path. It never opens or reads artifact content.
func inspectArtifact(root, rawURI string) (string, artifactState, error) {
	parsed, err := url.Parse(rawURI)
	if err != nil {
		return "", artifactSafe, fmt.Errorf("invalid artifact URI: %w", err)
	}
	if parsed.Scheme != "" || parsed.Host != "" || parsed.RawQuery != "" || parsed.Fragment != "" || strings.HasPrefix(rawURI, "//") {
		return "unsupported non-local URI", artifactUnknown, nil
	}
	decoded, err := url.PathUnescape(parsed.EscapedPath())
	if err != nil {
		return "", artifactSafe, fmt.Errorf("invalid percent encoding: %w", err)
	}
	if strings.ContainsRune(decoded, '\x00') {
		return "", artifactSafe, errors.New("artifact URI contains NUL")
	}
	if strings.Contains(decoded, `\`) || isWindowsDrivePath(decoded) || path.IsAbs(decoded) {
		return "unsupported absolute or non-POSIX URI", artifactUnknown, nil
	}

	normalized := path.Clean(decoded)
	if normalized == ".." || strings.HasPrefix(normalized, "../") {
		return normalized, artifactEscapes, nil
	}
	candidate := filepath.Join(root, filepath.FromSlash(normalized))
	if !isWithin(root, candidate) {
		return normalized, artifactEscapes, nil
	}

	info, err := os.Stat(candidate)
	if err == nil {
		canonical, err := filepath.EvalSymlinks(candidate)
		if err != nil {
			return "", artifactSafe, fmt.Errorf("resolve artifact path: %w", err)
		}
		if !isWithin(root, canonical) {
			return "", artifactSafe, errors.New("artifact path resolves outside root through a symlink")
		}
		if !info.Mode().IsRegular() {
			return normalized, artifactMissing, nil
		}
		return normalized, artifactSafe, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", artifactSafe, fmt.Errorf("inspect artifact path: %w", err)
	}
	if err := verifyExistingPrefix(root, candidate); err != nil {
		return "", artifactSafe, err
	}
	return normalized, artifactMissing, nil
}

func verifyExistingPrefix(root, candidate string) error {
	current := candidate
	for {
		if _, err := os.Lstat(current); err == nil {
			canonical, err := filepath.EvalSymlinks(current)
			if err != nil {
				return fmt.Errorf("resolve artifact path prefix: %w", err)
			}
			if !isWithin(root, canonical) {
				return errors.New("artifact path resolves outside root through a symlink")
			}
			return nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("inspect artifact path prefix: %w", err)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return errors.New("could not resolve artifact path inside root")
		}
		current = parent
	}
}

func isWithin(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func isWindowsDrivePath(value string) bool {
	return len(value) >= 2 && ((value[0] >= 'A' && value[0] <= 'Z') || (value[0] >= 'a' && value[0] <= 'z')) && value[1] == ':'
}

func intPointer(value int) *int {
	return &value
}

func diagnosticLocation(diagnostic Diagnostic) int {
	if diagnostic.Location == nil {
		return -1
	}
	return *diagnostic.Location
}

func readLog(path string) (sarifLog, error) {
	file, err := os.Open(path)
	if err != nil {
		return sarifLog{}, err
	}
	defer file.Close()
	if info, err := file.Stat(); err != nil {
		return sarifLog{}, err
	} else if info.Mode().IsRegular() && info.Size() > maxInputBytes {
		return sarifLog{}, fmt.Errorf("SARIF input exceeds %d bytes", maxInputBytes)
	}

	data, err := io.ReadAll(io.LimitReader(file, maxInputBytes+1))
	if err != nil {
		return sarifLog{}, err
	}
	if len(data) > maxInputBytes {
		return sarifLog{}, fmt.Errorf("SARIF input exceeds %d bytes", maxInputBytes)
	}
	if !utf8.Valid(data) {
		return sarifLog{}, errors.New("SARIF input is not valid UTF-8")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	var log sarifLog
	if err := decoder.Decode(&log); err != nil {
		return sarifLog{}, fmt.Errorf("invalid SARIF JSON: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return sarifLog{}, errors.New("invalid SARIF JSON: trailing document")
		}
		return sarifLog{}, fmt.Errorf("invalid SARIF JSON: %w", err)
	}
	if log.Version != "2.1.0" {
		return sarifLog{}, fmt.Errorf("unsupported SARIF version %q (expected 2.1.0)", log.Version)
	}
	return log, nil
}
