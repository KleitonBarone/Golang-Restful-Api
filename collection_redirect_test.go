package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAlbumCollectionRedirectIgnoresForwardedPrefix(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		t.Run(method, func(t *testing.T) {
			store := newAlbumStore(nil)
			router := setupRouterWithStore(store)
			const body = `{"id":"4","title":"Blue Train","artist":"John Coltrane","price":56.99}`
			request := httptest.NewRequest(method, "/albums/?limit=1&offset=0", strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("X-Forwarded-Prefix", "/untrusted")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			wantStatus := http.StatusMovedPermanently
			if method == http.MethodPost {
				wantStatus = http.StatusTemporaryRedirect
			}
			if response.Code != wantStatus {
				t.Fatalf("expected redirect status %d, got %d", wantStatus, response.Code)
			}
			if got := response.Header().Get("Location"); got != "/albums?limit=1&offset=0" {
				t.Fatalf("unexpected redirect location %q", got)
			}
			if len(store.list()) != 0 {
				t.Fatal("redirect changed the collection")
			}

			request = httptest.NewRequest(method, response.Header().Get("Location"), strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			response = httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if method == http.MethodGet {
				if response.Code != http.StatusOK || response.Body.String() != "[]" || response.Header().Get("X-Total-Count") != "0" {
					t.Fatalf("redirected list lost pagination: %d %s %v", response.Code, response.Body, response.Header())
				}
				return
			}
			var created album
			if response.Code != http.StatusCreated || json.Unmarshal(response.Body.Bytes(), &created) != nil || created.ID != "4" {
				t.Fatalf("redirected POST failed: %d %s", response.Code, response.Body)
			}
			if got, ok := store.get("4"); !ok || got != created {
				t.Fatalf("redirected POST did not store its body: %+v, found %v", got, ok)
			}
		})
	}
}
