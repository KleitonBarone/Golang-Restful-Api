package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestCanonicalAlbumURLsPreservePlusSigns(t *testing.T) {
	for _, id := range []string{"catalog+4", "catalog+/4", "catalog+ 4", "catalog%2F+/4"} {
		t.Run(id, func(t *testing.T) {
			other := album{ID: strings.ReplaceAll(id, "+", " "), Title: "Other", Artist: "Other", Price: 10}
			store := newAlbumStore([]album{other})
			router := setupRouterWithStore(store)
			created := album{ID: id, Title: "Original", Artist: "Artist", Price: 20}
			body, err := json.Marshal(created)
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodPost, "/albums", bytes.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != http.StatusCreated {
				t.Fatalf("create returned %d: %s", response.Code, response.Body)
			}
			location := response.Header().Get("Location")
			if want := "/albums/" + url.PathEscape(id); location != want {
				t.Fatalf("expected location %q, got %q", want, location)
			}

			replaced := created
			replaced.Title = "Replacement"
			body, err = json.Marshal(replaced)
			if err != nil {
				t.Fatal(err)
			}
			for _, step := range []struct {
				method string
				body   []byte
				status int
				title  string
			}{
				{method: http.MethodGet, status: http.StatusOK, title: "Original"},
				{method: http.MethodPut, body: body, status: http.StatusOK, title: "Replacement"},
				{method: http.MethodPatch, body: []byte(`{"title":"Patched"}`), status: http.StatusOK, title: "Patched"},
				{method: http.MethodDelete, status: http.StatusNoContent},
			} {
				request := httptest.NewRequest(step.method, location, bytes.NewReader(step.body))
				request.Header.Set("Content-Type", "application/json")
				response := httptest.NewRecorder()
				router.ServeHTTP(response, request)
				if response.Code != step.status {
					t.Fatalf("%s returned %d: %s", step.method, response.Code, response.Body)
				}
				if step.title != "" {
					var got album
					if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
						t.Fatal(err)
					}
					if got.ID != id || got.Title != step.title {
						t.Fatalf("%s returned the wrong album: %+v", step.method, got)
					}
				}
				if got, ok := store.get(other.ID); !ok || got != other {
					t.Fatalf("%s changed the other album: %+v", step.method, got)
				}
			}
			if _, ok := store.get(id); ok {
				t.Fatal("deleted album remains in storage")
			}
		})
	}
}
