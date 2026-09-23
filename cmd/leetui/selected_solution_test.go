package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Nano-AI/leetui/internal/auth"
	"github.com/Nano-AI/leetui/internal/leetcode"
)

const selectedCode = "class Solution:\n    def answer(self, n): return n + 1\n"

func selectedFixture(t *testing.T, filename string) (*app, string) {
	t.Helper()
	a := offlineContestApp(t)
	p := &leetcode.Problem{Slug: "selected-answer", FrontendID: "1", QuestionID: "101", Title: "Selected Answer",
		Content: "<pre>Output: 8</pre>", ExampleTestcases: "7",
		MetaData: `{"name":"answer","params":[{"name":"n","type":"integer"}],"return":{"type":"integer"}}`,
		Snippets: []leetcode.CodeSnippet{{LangSlug: "python3", Code: "class Solution: pass"}}}
	if err := a.store.SetDetail(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	for _, root := range []string{a.cfg.Workspace, t.TempDir()} {
		dir := filepath.Join(root, "0001-selected-answer")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		writeFixture(t, filepath.Join(dir, "solution.py"), "class Solution:\n    def answer(self, n): return -1\n")
		writeFixture(t, filepath.Join(dir, "testcases.txt"), "7\noutput:\n8\n")
		if root != a.cfg.Workspace {
			path := filepath.Join(dir, filename)
			writeFixture(t, path, selectedCode)
			return a, path
		}
	}
	panic("fixture missing")
}

func writeFixture(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRunUsesSelectedSolution(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 unavailable")
	}
	for _, shape := range []string{"file", "folder", "cwd", "alternate", "relative", "old-id"} {
		t.Run(shape, func(t *testing.T) {
			filename := "solution.py"
			if shape == "alternate" {
				filename = "my answer.py"
			}
			a, path := selectedFixture(t, filename)
			if shape == "old-id" {
				dir := filepath.Join(filepath.Dir(filepath.Dir(path)), "9999-selected-answer")
				if err := os.Rename(filepath.Dir(path), dir); err != nil {
					t.Fatal(err)
				}
				path = filepath.Join(dir, filename)
			}
			arg := path
			if shape == "relative" {
				t.Chdir(filepath.Dir(path))
				arg = filepath.Base(path)
			}
			if shape == "folder" {
				arg = filepath.Dir(path)
			}
			if shape == "cwd" {
				t.Chdir(filepath.Dir(path))
				arg = ""
			}
			stdout := os.Stdout
			output, err := os.CreateTemp(t.TempDir(), "stdout")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { os.Stdout = stdout; output.Close() })
			os.Stdout = output
			if code, err := runRun(a, []string{arg}); code != exitOK || err != nil {
				t.Fatalf("selected %s run = %d, %v", shape, code, err)
			}
			os.Stdout = stdout
			text, err := os.ReadFile(output.Name())
			if err != nil || !strings.Contains(string(text), path) {
				t.Errorf("submit hint lost the selected path: %s, %v", text, err)
			}
			got, err := os.ReadFile(path)
			if err != nil || string(got) != selectedCode {
				t.Fatalf("selected file changed: %q, %v", got, err)
			}
		})
	}
}

type captureSubmission struct {
	t              *testing.T
	endpoint, code string
}

func (c *captureSubmission) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.Method != "POST" || r.URL.Path != c.endpoint {
		return nil, fmt.Errorf("unexpected offline request: %s %s", r.Method, r.URL.Path)
	}
	var body struct {
		Code string `json:"typed_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		c.t.Fatal(err)
	}
	c.code = body.Code
	// Deliberately refuse the fake submission: no polling or accepted-commit path.
	return &http.Response{StatusCode: 400, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"error":"offline capture"}`)), Request: r}, nil
}

func TestSubmitUsesSelectedSolution(t *testing.T) {
	for _, contest := range []bool{false, true} {
		for _, shape := range []string{"file", "folder", "cwd", "alternate"} {
			t.Run(fmt.Sprintf("contest=%v/%s", contest, shape), func(t *testing.T) {
				filename := "solution.py"
				if shape == "alternate" {
					filename = "my answer.py"
				}
				a, path := selectedFixture(t, filename)
				capture := &captureSubmission{t: t, endpoint: "/problems/selected-answer/submit/"}
				if contest {
					capture.endpoint = "/contest/api/weekly-test/problems/selected-answer/submit/"
					if err := a.store.SetContest(context.Background(), &leetcode.Contest{
						ContestBrief: leetcode.ContestBrief{Slug: "weekly-test"},
						Questions:    []leetcode.ContestQuestion{{Slug: "selected-answer", QuestionID: "101"}},
					}); err != nil {
						t.Fatal(err)
					}
				}
				a.client = leetcode.New(leetcode.WithHTTPClient(&http.Client{Transport: capture}),
					leetcode.WithCredentials(auth.Credentials{Session: "offline-test", CSRF: "offline-test"}))
				arg := path
				if shape == "folder" {
					arg = filepath.Dir(path)
				}
				if shape == "cwd" {
					t.Chdir(filepath.Dir(path))
					arg = ""
				}
				if contest {
					_, _ = contestSubmit(a, "weekly-test", arg, "")
				} else {
					_, _ = runSubmit(a, []string{arg})
				}
				if strings.TrimSpace(capture.code) != strings.TrimSpace(selectedCode) {
					t.Fatalf("captured wrong source: %q", capture.code)
				}
			})
		}
	}
}
