package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// Exercise the served examples as requests, rather than checking generated text.
func TestOpenAPIExamplesSupportAlbumMutations(t *testing.T) {
	router := testRouter(t)
	document := httptest.NewRecorder()
	router.ServeHTTP(document, httptest.NewRequest(http.MethodGet, "/swagger/doc.json", nil))
	var spec struct {
		Definitions map[string]struct {
			Properties map[string]struct {
				Example json.RawMessage `json:"example"`
			} `json:"properties"`
		} `json:"definitions"`
	}
	if err := json.Unmarshal(document.Body.Bytes(), &spec); err != nil {
		t.Fatal(err)
	}
	example := func(schema string) []byte {
		t.Helper()
		properties := spec.Definitions[schema].Properties
		if len(properties) == 0 {
			t.Fatalf("missing schema %s", schema)
		}
		fields := make(map[string]json.RawMessage)
		for name, property := range properties {
			if len(property.Example) == 0 {
				t.Fatalf("missing example for %s.%s", schema, name)
			}
			fields[name] = property.Example
		}
		body, err := json.Marshal(fields)
		if err != nil {
			t.Fatal(err)
		}
		return body
	}
	createdBody := example("main.album")
	var created album
	if err := json.Unmarshal(createdBody, &created); err != nil {
		t.Fatal(err)
	}
	send := func(method, path string, body []byte, status int) album {
		t.Helper()
		request := httptest.NewRequest(method, path, bytes.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != status {
			t.Fatalf("%s %s: got %d, want %d: %s", method, path, response.Code, status, response.Body.String())
		}
		var result album
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		return result
	}
	if got := send(http.MethodPost, "/albums", createdBody, http.StatusCreated); got != created {
		t.Fatalf("created album = %+v, want %+v", got, created)
	}
	albumPath := "/albums/" + url.PathEscape(created.ID)
	if got := send(http.MethodPut, albumPath, createdBody, http.StatusOK); got != created {
		t.Fatalf("replaced album = %+v, want %+v", got, created)
	}
	patchBody := example("main.albumPatch")
	var patch albumPatch
	if err := json.Unmarshal(patchBody, &patch); err != nil {
		t.Fatal(err)
	}
	if want := patch.apply(created); want == created {
		t.Fatal("patch example must demonstrate a change to the album example")
	} else if got := send(http.MethodPatch, albumPath, patchBody, http.StatusOK); got != want {
		t.Fatalf("patched album = %+v, want %+v", got, want)
	}
}
