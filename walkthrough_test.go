package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// Execute the published commands so documentation and PowerShell behavior stay checked together.
func TestDocumentedAlbumWalkthrough(t *testing.T) {
	store := newAlbumStore(seedAlbums())
	server := httptest.NewServer(setupRouterWithStore(store))
	defer server.Close()
	results := runDocumentedWalkthrough(t, "docs/album-workflow.md", server.URL)
	if len(results) != 16 {
		t.Fatalf("expected 16 walkthrough results, got %d: %s", len(results), results)
	}
	created := album{ID: "walkthrough-demo", Title: "Kind of Blue", Artist: "Miles Davis", Price: 29.99}
	replaced := created
	replaced.Title, replaced.Price = "Kind of Blue (remastered)", 34.99
	patched := replaced
	patched.Price = 24.99
	wantAlbums := append([]album{created, created, replaced, patched}, seedAlbums()[:2]...)
	for i, want := range wantAlbums {
		var got album
		if err := json.Unmarshal(results[i+1], &got); err != nil || got != want {
			t.Errorf("walkthrough album result %d: got %s, want %+v, decode error: %v", i+1, results[i+1], want, err)
		}
	}
	var filtered album
	if err := json.Unmarshal(results[8], &filtered); err != nil || filtered != patched {
		t.Errorf("filtered walkthrough album: got %s, want %+v, decode error: %v", results[8], patched, err)
	}
	for i, want := range map[int]string{
		0: `{"status":"ok"}`, 7: `"4"`,
		9: `"1"`, 10: `200`, 11: `"1"`, 12: `0`,
		13: `204`, 14: `404`, 15: `{"message":"album not found"}`,
	} {
		if string(results[i]) != want {
			t.Errorf("walkthrough result %d: got %s, want %s", i, results[i], want)
		}
	}
	if got := store.list(); !reflect.DeepEqual(got, seedAlbums()) {
		t.Errorf("walkthrough must leave sample albums unchanged: %+v", got)
	}
}

func TestDocumentedOpenAPIWalkthrough(t *testing.T) {
	server := httptest.NewServer(setupRouter())
	defer server.Close()
	results := runDocumentedWalkthrough(t, "docs/openapi-workflow.md", server.URL)
	type responseMetadata struct {
		StatusCode int
		ETag       string
		BodyLength int
	}
	var got []responseMetadata
	for _, result := range results {
		var response responseMetadata
		if err := json.Unmarshal(result, &response); err != nil {
			t.Fatalf("decode OpenAPI walkthrough result: %v\n%s", err, result)
		}
		got = append(got, response)
	}
	if len(got) != 4 {
		t.Fatalf("expected four OpenAPI responses, got %d: %s", len(got), results)
	}
	wantETag := fmt.Sprintf(`"%x"`, sha256.Sum256(openAPIJSON))
	for i, wantStatus := range []int{200, 304, 304, 200} {
		wantLength := len(openAPIJSON)
		if wantStatus == 304 {
			wantLength = 0
		}
		if got[i].StatusCode != wantStatus || got[i].ETag != wantETag || got[i].BodyLength != wantLength {
			t.Errorf("OpenAPI response %d: got %+v, want status %d, ETag %s, body length %d",
				i, got[i], wantStatus, wantETag, wantLength)
		}
	}
}

func runDocumentedWalkthrough(t *testing.T, documentPath, baseURL string) []json.RawMessage {
	t.Helper()
	pwsh, err := exec.LookPath("pwsh")
	if err != nil {
		t.Fatalf("the documented walkthrough requires PowerShell 7 on PATH: %v", err)
	}
	document, err := os.ReadFile(documentPath)
	if err != nil {
		t.Fatal(err)
	}
	var workflow string
	for _, block := range strings.Split(strings.ReplaceAll(string(document), "\r\n", "\n"), "```powershell\n")[1:] {
		code, _, closed := strings.Cut(block, "```")
		if closed && strings.Contains(code, "$base = 'http://localhost:8080'") {
			if workflow != "" {
				t.Fatal("expected exactly one walkthrough block")
			}
			workflow = code
		}
	}
	if workflow == "" {
		t.Fatal("walkthrough block is missing")
	}
	workflow = strings.Replace(workflow, "$base = 'http://localhost:8080'", "$base = '"+baseURL+"'", 1)
	script := "$ErrorActionPreference = 'Stop'\n$ProgressPreference = 'SilentlyContinue'\n" +
		"$results = & {\n" + workflow + "\n}\nConvertTo-Json -InputObject @($results) -Depth 10 -Compress\n"
	path := filepath.Join(t.TempDir(), "walkthrough.ps1")
	if err := os.WriteFile(path, []byte(script), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, pwsh, "-NoLogo", "-NoProfile", "-NonInteractive", "-File", path)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	output, err := command.Output()
	if err != nil {
		t.Fatalf("walkthrough failed: %v\nstdout: %s\nstderr: %s", err, output, stderr.String())
	}
	var results []json.RawMessage
	if err := json.Unmarshal(output, &results); err != nil {
		t.Fatalf("decode walkthrough output: %v\n%s", err, output)
	}
	return results
}
