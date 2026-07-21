package preflight

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixture(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join("..", "..", "testdata", name, "results.sarif")
}

func fixtureRoot(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join("..", "..", "testdata", name)
}

func TestAnalyzeDiagnosticFixtures(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"missing-inline-message", "GSP001"},
		{"empty-artifact-uri", "GSP002"},
		{"unsupported-base-id", "GSP003"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			report, err := Analyze(fixtureRoot(t, test.name), "test", []string{fixture(t, test.name)})
			if err != nil {
				t.Fatal(err)
			}
			if len(report.Diagnostics) != 1 {
				t.Fatalf("diagnostics=%d, want 1: %#v", len(report.Diagnostics), report.Diagnostics)
			}
			if report.Diagnostics[0].ID != test.want {
				t.Fatalf("id=%q, want %q", report.Diagnostics[0].ID, test.want)
			}
			if report.Diagnostics[0].Run != 0 || report.Diagnostics[0].Result != 0 {
				t.Fatalf("unexpected indexes: %#v", report.Diagnostics[0])
			}
		})
	}
}

func TestAnalyzeSafeSrcroot(t *testing.T) {
	report, err := Analyze(fixtureRoot(t, "safe-srcroot"), "test", []string{fixture(t, "safe-srcroot")})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", report.Diagnostics)
	}
	if report.Summary.Results != 1 {
		t.Fatalf("results=%d, want 1", report.Summary.Results)
	}
}

func TestAnalyzeRejectsInvalidJSON(t *testing.T) {
	_, err := Analyze(fixtureRoot(t, "invalid-json"), "test", []string{fixture(t, "invalid-json")})
	if err == nil {
		t.Fatal("expected invalid JSON error")
	}
}

func TestAnalyzeRepositoryPaths(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"root-escape", "GSP004"},
		{"missing-checkout-file", "GSP005"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			report, err := Analyze(fixtureRoot(t, test.name), "test", []string{fixture(t, test.name)})
			if err != nil {
				t.Fatal(err)
			}
			if len(report.Diagnostics) != 1 || report.Diagnostics[0].ID != test.want {
				t.Fatalf("diagnostics=%#v, want %s", report.Diagnostics, test.want)
			}
			if test.name == "root-escape" && report.Diagnostics[0].Path != "../../outside.tf" {
				t.Fatalf("normalized path=%q, want ../../outside.tf", report.Diagnostics[0].Path)
			}
		})
	}
}

func TestAnalyzeMarksDirectoryAsNotRegular(t *testing.T) {
	temp := t.TempDir()
	root := filepath.Join(temp, "root")
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	sarif := `{"version":"2.1.0","runs":[{"results":[{"message":{"text":"safe"},"locations":[{"physicalLocation":{"artifactLocation":{"uri":"src"}}}]}]}]}`
	input := filepath.Join(temp, "input.sarif")
	if err := os.WriteFile(input, []byte(sarif), 0o600); err != nil {
		t.Fatal(err)
	}
	report, err := Analyze(root, "test", []string{input})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Diagnostics) != 1 || report.Diagnostics[0].ID != "GSP005" {
		t.Fatalf("unexpected diagnostics: %#v", report.Diagnostics)
	}
}

func TestAnalyzePercentEncodedUnicodePath(t *testing.T) {
	report, err := Analyze(fixtureRoot(t, "safe-unicode"), "test", []string{fixture(t, "safe-unicode")})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Diagnostics) != 0 || len(report.Unknowns) != 0 {
		t.Fatalf("unexpected report: %#v", report)
	}
}

func TestAnalyzeClassifiesUnsupportedURIsAsUnknown(t *testing.T) {
	report, err := Analyze(fixtureRoot(t, "unsupported-uri"), "test", []string{fixture(t, "unsupported-uri")})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Diagnostics) != 0 || len(report.Unknowns) != 4 || report.Summary.Unknowns != 4 {
		t.Fatalf("unexpected report: %#v", report)
	}
}

func TestAnalyzeRejectsSymlinkEscape(t *testing.T) {
	temp := t.TempDir()
	root := filepath.Join(temp, "root")
	outside := filepath.Join(temp, "outside")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("must not be read"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	sarif := `{"version":"2.1.0","runs":[{"results":[{"message":{"text":"safe"},"locations":[{"physicalLocation":{"artifactLocation":{"uri":"link/secret.txt"}}}]}]}]}`
	input := filepath.Join(temp, "input.sarif")
	if err := os.WriteFile(input, []byte(sarif), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Analyze(root, "test", []string{input})
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("error=%v, want symlink escape", err)
	}
}

func TestAnalyzeRejectsBoundariesBeforeDiagnostics(t *testing.T) {
	tests := []struct {
		name    string
		content []byte
		want    string
	}{
		{"invalid-utf8", []byte("{\"version\":\"2.1.0\",\"runs\":[]}" + string([]byte{0xff})), "not valid UTF-8"},
		{"wrong-version", []byte(`{"version":"2.0.0","runs":[]}`), "unsupported SARIF version"},
		{"invalid-percent", []byte(`{"version":"2.1.0","runs":[{"results":[{"message":{"text":"safe"},"locations":[{"physicalLocation":{"artifactLocation":{"uri":"src/%ZZ.js"}}}]}]}]}`), "invalid artifact URI"},
		{"long-uri", []byte(`{"version":"2.1.0","runs":[{"results":[{"message":{"text":"safe"},"locations":[{"physicalLocation":{"artifactLocation":{"uri":"` + strings.Repeat("a", maxArtifactURIBytes+1) + `"}}}]}]}]}`), "artifact URI exceeds"},
		{"long-base-id", []byte(`{"version":"2.1.0","runs":[{"results":[{"message":{"text":"safe"},"locations":[{"physicalLocation":{"artifactLocation":{"uri":"safe.go","uriBaseId":"` + strings.Repeat("B", maxURIBaseIDBytes+1) + `"}}}]}]}]}`), "uriBaseId exceeds"},
		{"long-rule-id", []byte(`{"version":"2.1.0","runs":[{"results":[{"ruleId":"` + strings.Repeat("R", maxRuleIDBytes+1) + `","message":{"text":"safe"}}]}]}`), "rule ID exceeds"},
		{"too-many-runs", []byte(`{"version":"2.1.0","runs":[` + strings.Repeat(`{},`, maxRuns) + `{}` + `]}`), "SARIF runs"},
		{"too-many-results", []byte(`{"version":"2.1.0","runs":[{"results":[` + strings.Repeat(`{},`, maxResults) + `{}` + `]}]}`), "SARIF results"},
		{"too-many-locations", []byte(`{"version":"2.1.0","runs":[{"results":[{"message":{"text":"safe"},"locations":[` + strings.Repeat(`{},`, maxLocations) + `{}` + `]}]}]}`), "SARIF locations"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			temp := t.TempDir()
			input := filepath.Join(temp, "input.sarif")
			if err := os.WriteFile(input, test.content, 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := Analyze(temp, "test", []string{input})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error=%v, want substring %q", err, test.want)
			}
		})
	}
}

func TestAnalyzeRejectsTooManyInputs(t *testing.T) {
	inputs := make([]string, 33)
	_, err := Analyze(t.TempDir(), "test", inputs)
	if err == nil || !strings.Contains(err.Error(), "at most 32") {
		t.Fatalf("error=%v, want input count rejection", err)
	}
}

func TestAnalyzeRejectsOversizedAndUnreadableInput(t *testing.T) {
	temp := t.TempDir()
	oversized := filepath.Join(temp, "oversized.sarif")
	if err := os.WriteFile(oversized, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(oversized, maxInputBytes+1); err != nil {
		t.Fatal(err)
	}
	if _, err := Analyze(temp, "test", []string{oversized}); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("oversized error=%v", err)
	}

	unreadable := filepath.Join(temp, "unreadable.sarif")
	if err := os.WriteFile(unreadable, []byte(`{"version":"2.1.0","runs":[]}`), 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(unreadable, 0o600) })
	if _, err := Analyze(temp, "test", []string{unreadable}); err == nil {
		t.Fatal("expected unreadable input error")
	}
}
