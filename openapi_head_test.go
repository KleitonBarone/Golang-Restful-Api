package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAPIDocumentsBodylessHEADResponses(t *testing.T) {
	response := httptest.NewRecorder()
	testRouter(t).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/swagger/doc.json", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("OpenAPI document returned status %d", response.Code)
	}

	var spec struct {
		Paths map[string]struct {
			Head struct {
				Responses map[string]struct {
					Schema  json.RawMessage            `json:"schema"`
					Headers map[string]json.RawMessage `json:"headers"`
				} `json:"responses"`
			} `json:"head"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &spec); err != nil {
		t.Fatal(err)
	}

	for _, route := range []struct {
		path     string
		statuses []string
	}{
		{path: "/albums", statuses: []string{"200", "400"}},
		{path: "/albums/{id}", statuses: []string{"200", "404"}},
		{path: "/health", statuses: []string{"200"}},
	} {
		t.Run(route.path, func(t *testing.T) {
			responses := spec.Paths[route.path].Head.Responses
			for _, status := range route.statuses {
				documented, ok := responses[status]
				if !ok {
					t.Errorf("missing HEAD status %s", status)
				}
				if len(documented.Schema) != 0 {
					t.Errorf("HEAD status %s documents a response body: %s", status, documented.Schema)
				}
			}
		})
	}
	for _, header := range []string{"Link", "X-Total-Count"} {
		if _, ok := spec.Paths["/albums"].Head.Responses["200"].Headers[header]; !ok {
			t.Errorf("HEAD /albums is missing the %s response header", header)
		}
	}
}
