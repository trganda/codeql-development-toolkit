package bundle

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"
)

// ValidateOptions configures Validate.
type ValidateOptions struct {
	Expected string
	Actual   string
}

// Validate compares the result findings of two SARIF files. It is intended for
// use in bundle integration tests, where the actual analysis output is checked
// against a checked-in expected SARIF.
//
// Comparison is restricted to the semantic findings (ruleId, message text, and
// physical locations) — telemetry, timestamps, tool versions, and absolute
// machine paths are intentionally ignored, as they vary between runs.
func Validate(opts *ValidateOptions) error {
	expected, err := readSarifResults(opts.Expected)
	if err != nil {
		return fmt.Errorf("read expected: %w", err)
	}
	actual, err := readSarifResults(opts.Actual)
	if err != nil {
		return fmt.Errorf("read actual: %w", err)
	}

	missing, extra := diffResults(expected, actual)
	if len(missing) == 0 && len(extra) == 0 {
		slog.Info("SARIF results match", "count", len(expected))
		return nil
	}

	slog.Error("SARIF results differ",
		"expected", len(expected), "actual", len(actual),
		"missing", len(missing), "unexpected", len(extra))
	for _, k := range missing {
		slog.Error("Missing result (in expected, not in actual)", "result", k)
	}
	for _, k := range extra {
		slog.Error("Unexpected result (in actual, not in expected)", "result", k)
	}
	return fmt.Errorf("SARIF results differ: %d missing, %d unexpected", len(missing), len(extra))
}

type sarifFile struct {
	Runs []struct {
		Results []sarifResult `json:"results"`
	} `json:"runs"`
}

type sarifResult struct {
	RuleID  string `json:"ruleId"`
	Message struct {
		Text string `json:"text"`
	} `json:"message"`
	Locations []struct {
		PhysicalLocation struct {
			ArtifactLocation struct {
				URI       string `json:"uri"`
				URIBaseID string `json:"uriBaseId"`
			} `json:"artifactLocation"`
			Region struct {
				StartLine   int `json:"startLine"`
				StartColumn int `json:"startColumn"`
				EndLine     int `json:"endLine"`
				EndColumn   int `json:"endColumn"`
			} `json:"region"`
		} `json:"physicalLocation"`
	} `json:"locations"`
}

func readSarifResults(path string) ([]sarifResult, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("path is empty")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s sarifFile
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	var out []sarifResult
	for _, run := range s.Runs {
		out = append(out, run.Results...)
	}
	sortResults(out)
	return out, nil
}

func sortResults(rs []sarifResult) {
	sort.SliceStable(rs, func(i, j int) bool {
		return resultKey(rs[i]) < resultKey(rs[j])
	})
}

func resultKey(r sarifResult) string {
	var b strings.Builder
	b.WriteString(r.RuleID)
	b.WriteString("|")
	b.WriteString(r.Message.Text)
	for _, loc := range r.Locations {
		p := loc.PhysicalLocation
		fmt.Fprintf(&b, "|%s@%d:%d-%d:%d",
			p.ArtifactLocation.URI,
			p.Region.StartLine, p.Region.StartColumn,
			p.Region.EndLine, p.Region.EndColumn,
		)
	}
	return b.String()
}

func diffResults(expected, actual []sarifResult) (missing, extra []string) {
	expectedSet := map[string]int{}
	for _, r := range expected {
		expectedSet[resultKey(r)]++
	}
	actualSet := map[string]int{}
	for _, r := range actual {
		actualSet[resultKey(r)]++
	}

	for k, n := range expectedSet {
		if actualSet[k] < n {
			for i := 0; i < n-actualSet[k]; i++ {
				missing = append(missing, k)
			}
		}
	}
	for k, n := range actualSet {
		if expectedSet[k] < n {
			for i := 0; i < n-expectedSet[k]; i++ {
				extra = append(extra, k)
			}
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	return missing, extra
}
