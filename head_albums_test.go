package main

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

func TestHeadAlbumsMatchesGetMetadata(t *testing.T) {
	for _, scenario := range []struct {
		path   string
		status int
		total  string
	}{
		{path: "/albums", status: http.StatusOK},
		{path: "/albums?limit=1&offset=1", status: http.StatusOK, total: "3"},
		{path: "/albums?offset=99", status: http.StatusOK, total: "3"},
		{path: "/albums?limit=0", status: http.StatusBadRequest},
		{path: "/albums?offset=-1", status: http.StatusBadRequest},
		{path: "/albums?limit=1&limit=2", status: http.StatusBadRequest},
		{path: "/albums?limit=%zz", status: http.StatusBadRequest},
		{path: "/albums/1", status: http.StatusOK},
		{path: "/albums/missing", status: http.StatusNotFound},
		{path: "/albums/catalog+%2F4", status: http.StatusOK},
	} {
		t.Run(scenario.path, func(t *testing.T) {
			initial := append(seedAlbums()[:2], album{ID: "catalog+/4", Title: "Escaped", Artist: "Artist", Price: 10})
			store := newAlbumStore(initial)
			router := setupRouterWithStore(store)
			get := httptest.NewRecorder()
			router.ServeHTTP(get, httptest.NewRequest(http.MethodGet, scenario.path, nil))
			head := httptest.NewRecorder()
			router.ServeHTTP(head, httptest.NewRequest(http.MethodHead, scenario.path, nil))
			if head.Code != scenario.status || head.Code != get.Code {
				t.Fatalf("expected status %d, got HEAD %d and GET %d", scenario.status, head.Code, get.Code)
			}
			if head.Body.Len() != 0 {
				t.Fatalf("HEAD returned a body: %s", head.Body)
			}
			if get.Body.Len() == 0 {
				t.Fatal("GET lost its response body")
			}
			if got := head.Header().Get("Content-Type"); got != "application/json; charset=utf-8" || got != get.Header().Get("Content-Type") {
				t.Fatalf("HEAD content type %q differs from GET %q", got, get.Header().Get("Content-Type"))
			}
			if got := head.Header().Get("X-Total-Count"); got != scenario.total || got != get.Header().Get("X-Total-Count") {
				t.Fatalf("expected total %q, got HEAD %q and GET %q", scenario.total, got, get.Header().Get("X-Total-Count"))
			}
			if !slices.Equal(store.list(), initial) {
				t.Fatal("read requests changed stored albums")
			}
		})
	}
}

func TestHeadAlbumsEmptyCollection(t *testing.T) {
	router := setupRouterWithStore(newAlbumStore(nil))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodHead, "/albums?limit=1", nil))
	if response.Code != http.StatusOK || response.Body.Len() != 0 || response.Header().Get("X-Total-Count") != "0" {
		t.Fatalf("expected empty HEAD page with total zero, got %d %s %v", response.Code, response.Body, response.Header())
	}
}

func TestHeadAlbumRedirectsPreserveTarget(t *testing.T) {
	router := setupRouterWithStore(newAlbumStore([]album{{ID: "catalog+/4", Title: "Escaped", Artist: "Artist", Price: 10}}))
	for _, path := range []string{"/albums", "/albums/catalog+%2F4"} {
		t.Run(path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodHead, path+"/?limit=1&label=a%2Bb", nil)
			request.Header.Set("X-Forwarded-Prefix", "/untrusted")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != http.StatusTemporaryRedirect || response.Body.Len() != 0 {
				t.Fatalf("expected bodyless 307, got %d %s", response.Code, response.Body)
			}
			if got, want := response.Header().Get("Location"), path+"?limit=1&label=a%2Bb"; got != want {
				t.Fatalf("expected location %q, got %q", want, got)
			}
			followed := httptest.NewRecorder()
			router.ServeHTTP(followed, httptest.NewRequest(http.MethodHead, response.Header().Get("Location"), nil))
			if followed.Code != http.StatusOK || followed.Body.Len() != 0 {
				t.Fatalf("redirected HEAD returned %d %s", followed.Code, followed.Body)
			}
		})
	}
}
