package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixture(name string) string {
	return filepath.Join("..", "..", "testdata", name, "results.sarif")
}

func TestRunExitContract(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantCode int
		wantText string
	}{
		{"safe", []string{"check", "--root", filepath.Dir(fixture("safe-srcroot")), fixture("safe-srcroot")}, 0, "diagnostics=0"},
		{"diagnostic", []string{"check", "--root", filepath.Dir(fixture("missing-inline-message")), fixture("missing-inline-message")}, 1, "GSP001"},
		{"invalid", []string{"check", "--root", filepath.Dir(fixture("invalid-json")), fixture("invalid-json")}, 2, "input error:"},
		{"unknown-only", []string{"check", "--root", filepath.Dir(fixture("unsupported-uri")), fixture("unsupported-uri")}, 0, "unknowns=4"},
		{"bad-format", []string{"check", "--format", "xml", fixture("safe-srcroot")}, 2, "format must be text or json"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Run(test.args, &stdout, &stderr, "test")
			if code != test.wantCode {
				t.Fatalf("code=%d want=%d stdout=%q stderr=%q", code, test.wantCode, stdout.String(), stderr.String())
			}
			if !strings.Contains(stdout.String()+stderr.String(), test.wantText) {
				t.Fatalf("output does not contain %q: stdout=%q stderr=%q", test.wantText, stdout.String(), stderr.String())
			}
		})
	}
}

func TestRunHelpContract(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "long option",
			args: []string{"--help"},
			want: []string{"Usage:", "check", "version", "GSP001", "Exit codes:"},
		},
		{
			name: "short option",
			args: []string{"-h"},
			want: []string{"Usage:", "check", "version", "GSP005", "Exit codes:"},
		},
		{
			name: "help command",
			args: []string{"help"},
			want: []string{"Usage:", "consumer profile", "current checkout", "Exit codes:"},
		},
		{
			name: "check long option",
			args: []string{"check", "--help"},
			want: []string{"Usage:", "--root", "--format", "GSP001", "GSP005", "Exit codes:"},
		},
		{
			name: "check short option",
			args: []string{"check", "-h"},
			want: []string{"Usage:", "--root", "--format", "GSP001", "GSP005", "Exit codes:"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Run(test.args, &stdout, &stderr, "test")
			if code != 0 || stderr.Len() != 0 {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
			for _, want := range test.want {
				if !strings.Contains(stdout.String(), want) {
					t.Fatalf("stdout does not contain %q: %q", want, stdout.String())
				}
			}
		})
	}
}

func TestRunJSONContract(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"check", "--root", filepath.Dir(fixture("unsupported-base-id")), "--format", "json", fixture("unsupported-base-id")}, &stdout, &stderr, "test")
	if code != 1 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	var report struct {
		SchemaVersion int `json:"schemaVersion"`
		Diagnostics   []struct {
			ID string `json:"id"`
		} `json:"diagnostics"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.SchemaVersion != 1 || len(report.Diagnostics) != 1 || report.Diagnostics[0].ID != "GSP003" {
		t.Fatalf("unexpected report: %#v", report)
	}
}

func TestRunMultiRunGoldenContracts(t *testing.T) {
	input := fixture("multi-run")
	root := filepath.Dir(input)
	tests := []struct {
		name   string
		format string
		golden string
	}{
		{"text", "text", "expected.text"},
		{"json", "json", "expected.json"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Run([]string{"check", "--root", root, "--format", test.format, input}, &stdout, &stderr, "test")
			if code != 1 || stderr.Len() != 0 {
				t.Fatalf("code=%d stderr=%q", code, stderr.String())
			}
			want, err := os.ReadFile(filepath.Join(root, test.golden))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(stdout.Bytes(), want) {
				t.Fatalf("output mismatch\n--- got ---\n%s--- want ---\n%s", stdout.Bytes(), want)
			}
		})
	}
}
