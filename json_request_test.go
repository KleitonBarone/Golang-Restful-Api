package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

func TestAlbumMutationsValidateUnicodeEscapes(t *testing.T) {
	for _, route := range []struct {
		method string
		path   string
		body   string
		status int
		id     string
	}{
		{http.MethodPost, "/albums", `{"id":"4","title":%s,"artist":"Miles Davis","price":29.99}`, http.StatusCreated, "4"},
		{http.MethodPut, "/albums/2", `{"id":"2","title":%s,"artist":"Gerry Mulligan","price":24.99}`, http.StatusOK, "2"},
		{http.MethodPatch, "/albums/2", `{"title":%s}`, http.StatusOK, "2"},
	} {
		for _, scenario := range []struct {
			name  string
			title string
			want  string
		}{
			{name: "lone high surrogate", title: `"\ud800"`},
			{name: "lone low surrogate", title: `"\udfff"`},
			{name: "reversed pair", title: `"\udc00\ud800"`},
			{name: "two high surrogates", title: `"\ud800\udbff"`},
			{name: "high followed by BMP escape", title: `"\ud800\u0041"`},
			{name: "separated pair", title: `"\ud800x\udc00"`},
			{name: "escaped second backslash", title: `"\ud800\\udc00"`},
			{name: "valid pair", title: `"\uD834\uDD1E"`, want: "𝄞"},
			{name: "pair boundaries", title: `"\ud800\udc00\udbff\udfff"`, want: "\U00010000\U0010ffff"},
			{name: "BMP boundaries", title: `"\ud7ff\ue000"`, want: "\ud7ff\ue000"},
			{name: "literal escape text", title: `"\\ud800"`, want: `\ud800`},
			{name: "backslash before pair", title: `"\\\ud834\udd1e"`, want: "\\𝄞"},
			{name: "intentional replacement character", title: `"\ufffd"`, want: "�"},
		} {
			t.Run(route.method+"/"+scenario.name, func(t *testing.T) {
				store := newAlbumStore(seedAlbums())
				before := store.list()
				request := httptest.NewRequest(route.method, route.path, bytes.NewBufferString(fmt.Sprintf(route.body, scenario.title)))
				request.Header.Set("Content-Type", "application/json")
				response := httptest.NewRecorder()
				setupRouterWithStore(store).ServeHTTP(response, request)

				if scenario.want == "" {
					if response.Code != http.StatusBadRequest {
						t.Fatalf("expected status 400, got %d: %s", response.Code, response.Body)
					}
					var got errorResponse
					if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil || got.Message != "invalid request body" {
						t.Fatalf("expected invalid-body response, got %s, error %v", response.Body, err)
					}
					if !slices.Equal(before, store.list()) {
						t.Fatal("invalid Unicode escape changed stored albums")
					}
					return
				}
				if response.Code != route.status {
					t.Fatalf("expected status %d, got %d: %s", route.status, response.Code, response.Body)
				}
				if got, ok := store.get(route.id); !ok || got.Title != scenario.want {
					t.Fatalf("expected stored title %q, got %#v", scenario.want, got)
				}
			})
		}
	}
}
