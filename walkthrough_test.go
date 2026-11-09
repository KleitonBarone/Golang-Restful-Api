package main

import (
	"bytes"
	"context"
	"encoding/json"
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
	pwsh, err := exec.LookPath("pwsh")
	if err != nil {
		t.Fatalf("the documented walkthrough requires PowerShell 7 on PATH: %v", err)
	}
	document, err := os.ReadFile("docs/album-workflow.md")
	if err != nil {
		t.Fatal(err)
	}
	var workflow string
	for _, block := range strings.Split(strings.ReplaceAll(string(document), "\r\n", "\n"), "```powershell\n")[1:] {
		code, _, closed := strings.Cut(block, "```")
		if closed && strings.Contains(code, "$base = 'http://localhost:8080'") {
			if workflow != "" {
				t.Fatal("expected exactly one album walkthrough block")
			}
			workflow = code
		}
	}
	if workflow == "" {
		t.Fatal("album walkthrough block is missing")
	}
	store := newAlbumStore(seedAlbums())
	server := httptest.NewServer(setupRouterWithStore(store))
	defer server.Close()
	workflow = strings.Replace(workflow, "$base = 'http://localhost:8080'", "$base = '"+server.URL+"'", 1)
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
	if len(results) != 11 {
		t.Fatalf("expected 11 walkthrough results, got %d: %s", len(results), output)
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
	for i, want := range map[int]string{0: `{"status":"ok"}`, 7: `"4"`, 8: `204`, 9: `404`, 10: `{"message":"album not found"}`} {
		if string(results[i]) != want {
			t.Errorf("walkthrough result %d: got %s, want %s", i, results[i], want)
		}
	}
	if got := store.list(); !reflect.DeepEqual(got, seedAlbums()) {
		t.Errorf("walkthrough must leave sample albums unchanged: %+v", got)
	}
}
