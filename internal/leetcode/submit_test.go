package leetcode

import (
	"encoding/json"
	"testing"
)

// The judge sends code_output two different ways depending on which endpoint answered,
// and a []string field decodes one and fails the other. That failure reached a user as
// "Submit failed: … cannot unmarshal string into Go struct field" on a submission the
// judge had actually accepted — the answer was fine, the decode was not.

func TestJudgementDecodesBothShapesOfCodeOutput(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want []string
	}{
		{
			// interpret_solution — running against the examples.
			"run returns a list",
			`{"state":"SUCCESS","status_code":10,"code_output":["[0,1]","[1,2]"]}`,
			[]string{"[0,1]", "[1,2]"},
		},
		{
			// submit — the shape that was failing.
			"submit returns an empty string",
			`{"state":"SUCCESS","status_code":10,"code_output":""}`,
			nil,
		},
		{
			"submit returns a string with content",
			`{"state":"SUCCESS","status_code":11,"code_output":"line one\nline two"}`,
			[]string{"line one", "line two"},
		},
		{
			"null",
			`{"state":"SUCCESS","status_code":10,"code_output":null}`,
			nil,
		},
		{
			"absent",
			`{"state":"SUCCESS","status_code":10}`,
			nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var j Judgement
			if err := json.Unmarshal([]byte(tc.body), &j); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if len(j.CodeOutput) != len(tc.want) {
				t.Fatalf("CodeOutput = %q, want %q", j.CodeOutput, tc.want)
			}
			for i := range tc.want {
				if j.CodeOutput[i] != tc.want[i] {
					t.Errorf("line %d = %q, want %q", i, j.CodeOutput[i], tc.want[i])
				}
			}
		})
	}
}

// TestAcceptedSubmissionDecodes is the reported failure end to end: a real Accepted
// payload must decode, and must read as accepted rather than as an error.
func TestAcceptedSubmissionDecodes(t *testing.T) {
	body := `{
		"status_code": 10,
		"lang": "cpp",
		"run_success": true,
		"status_runtime": "58 ms",
		"memory": 20238000,
		"code_output": "",
		"std_output": "",
		"last_testcase": "",
		"expected_output": "",
		"total_correct": 63,
		"total_testcases": 63,
		"runtime_percentile": 91.23,
		"status_memory": "20.2 MB",
		"memory_percentile": 44.1,
		"state": "SUCCESS",
		"status_msg": "Accepted",
		"submission_id": "2097544646"
	}`

	var j Judgement
	if err := json.Unmarshal([]byte(body), &j); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !j.Accepted() {
		t.Errorf("Accepted() = false, want true (status_msg %q)", j.StatusMsg)
	}
	if !j.Done() {
		t.Error("Done() = false on a SUCCESS state")
	}
	if j.Runtime != "58 ms" || j.RuntimePercentile != 91.23 {
		t.Errorf("runtime = %q / %v", j.Runtime, j.RuntimePercentile)
	}
	if j.SubmissionID.String() != "2097544646" {
		t.Errorf("submission id = %q", j.SubmissionID)
	}
	if s := j.CodeOutput.String(); s != "" {
		// An empty string is no output, not one blank line.
		t.Errorf("CodeOutput.String() = %q, want empty", s)
	}
}
