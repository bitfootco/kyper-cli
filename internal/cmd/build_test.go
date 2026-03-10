package cmd

import (
	"encoding/json"
	"testing"
)

func TestParseTrivyJSON(t *testing.T) {
	tests := []struct {
		name            string
		json            string
		ignored         map[string]bool
		wantStatus      string
		wantFixCrit     int
		wantFixHigh     int
		wantUnfixed     int
		wantIgnored     int
		wantBlockingLen int
		wantErr         bool
	}{
		{
			name:       "clean scan",
			json:       `{"Results": [{"Vulnerabilities": []}]}`,
			ignored:    map[string]bool{},
			wantStatus: "passed",
		},
		{
			name: "fixable CRITICAL fails",
			json: `{"Results": [{"Vulnerabilities": [
				{"VulnerabilityID": "CVE-2025-001", "PkgName": "openssl", "Severity": "CRITICAL", "FixedVersion": "3.0.15"}
			]}]}`,
			ignored:         map[string]bool{},
			wantStatus:      "failed",
			wantFixCrit:     1,
			wantBlockingLen: 1,
		},
		{
			name: "only unfixed passes with advisories",
			json: `{"Results": [{"Vulnerabilities": [
				{"VulnerabilityID": "CVE-2025-001", "PkgName": "libc", "Severity": "HIGH", "FixedVersion": ""},
				{"VulnerabilityID": "CVE-2025-002", "PkgName": "glibc", "Severity": "CRITICAL", "FixedVersion": ""}
			]}]}`,
			ignored:     map[string]bool{},
			wantStatus:  "passed_with_advisories",
			wantUnfixed: 2,
		},
		{
			name: "mix fixable and unfixed fails with advisory counted separately",
			json: `{"Results": [{"Vulnerabilities": [
				{"VulnerabilityID": "CVE-2025-001", "PkgName": "openssl", "Severity": "CRITICAL", "FixedVersion": "3.0.15"},
				{"VulnerabilityID": "CVE-2025-002", "PkgName": "libc", "Severity": "HIGH", "FixedVersion": ""}
			]}]}`,
			ignored:         map[string]bool{},
			wantStatus:      "failed",
			wantFixCrit:     1,
			wantUnfixed:     1,
			wantBlockingLen: 1,
		},
		{
			name: "all ignored passes with ignores",
			json: `{"Results": [{"Vulnerabilities": [
				{"VulnerabilityID": "CVE-2025-001", "PkgName": "curl", "Severity": "HIGH", "FixedVersion": "8.0"}
			]}]}`,
			ignored:     map[string]bool{"CVE-2025-001": true},
			wantStatus:  "passed_with_ignores",
			wantIgnored: 1,
		},
		{
			name:    "invalid JSON returns error",
			json:    "not json at all",
			ignored: map[string]bool{},
			wantErr: true,
		},
		{
			name:       "empty results",
			json:       `{"Results": []}`,
			ignored:    map[string]bool{},
			wantStatus: "passed",
		},
		{
			name: "unfixed plus ignored with no fixable passes with advisories",
			json: `{"Results": [{"Vulnerabilities": [
				{"VulnerabilityID": "CVE-2025-001", "PkgName": "libc", "Severity": "HIGH", "FixedVersion": ""},
				{"VulnerabilityID": "CVE-2025-002", "PkgName": "curl", "Severity": "HIGH", "FixedVersion": "8.0"}
			]}]}`,
			ignored:     map[string]bool{"CVE-2025-002": true},
			wantStatus:  "passed_with_advisories",
			wantUnfixed: 1,
			wantIgnored: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sr, err := parseTrivyJSON([]byte(tt.json), tt.ignored)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if sr.Status != tt.wantStatus {
				t.Errorf("status: got %q, want %q", sr.Status, tt.wantStatus)
			}
			if sr.FixableCritical != tt.wantFixCrit {
				t.Errorf("fixable_critical: got %d, want %d", sr.FixableCritical, tt.wantFixCrit)
			}
			if sr.FixableHigh != tt.wantFixHigh {
				t.Errorf("fixable_high: got %d, want %d", sr.FixableHigh, tt.wantFixHigh)
			}
			if sr.UnfixedCount != tt.wantUnfixed {
				t.Errorf("unfixed_count: got %d, want %d", sr.UnfixedCount, tt.wantUnfixed)
			}
			if sr.IgnoredCount != tt.wantIgnored {
				t.Errorf("ignored_count: got %d, want %d", sr.IgnoredCount, tt.wantIgnored)
			}
			if tt.wantBlockingLen > 0 && len(sr.Blocking) != tt.wantBlockingLen {
				t.Errorf("blocking len: got %d, want %d", len(sr.Blocking), tt.wantBlockingLen)
			}
		})
	}
}

func TestParseTrivyJSONOutput(t *testing.T) {
	sr := &scanResult{
		Image:           "kyper-local/test:1.0.0",
		Status:          "failed",
		FixableCritical: 1,
		FixableHigh:     0,
		UnfixedCount:    2,
		IgnoredCount:    0,
		Blocking:        []scanVuln{{ID: "CVE-2025-001", Pkg: "openssl", Severity: "CRITICAL", Fixed: "3.0.15"}},
		Advisory:        []scanVuln{{ID: "CVE-2025-002", Pkg: "libc", Severity: "HIGH"}, {ID: "CVE-2025-003", Pkg: "glibc", Severity: "HIGH"}},
		Ignored:         nil,
	}

	data, err := json.Marshal(sr)
	if err != nil {
		t.Fatalf("json marshal failed: %v", err)
	}

	var decoded scanResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}

	if decoded.Status != "failed" {
		t.Errorf("status roundtrip: got %q, want %q", decoded.Status, "failed")
	}
	if decoded.FixableCritical != 1 {
		t.Errorf("fixable_critical roundtrip: got %d, want %d", decoded.FixableCritical, 1)
	}
	if len(decoded.Blocking) != 1 {
		t.Errorf("blocking roundtrip: got %d, want %d", len(decoded.Blocking), 1)
	}
	if len(decoded.Advisory) != 2 {
		t.Errorf("advisory roundtrip: got %d, want %d", len(decoded.Advisory), 2)
	}
}
