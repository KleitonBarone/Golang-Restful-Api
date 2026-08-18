package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
)

func testRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	return setupRouterWithStore(newAlbumStore(seedAlbums()))
}

type stubAlbumStore struct {
	albums []album
}

func (s *stubAlbumStore) list() []album {
	return append([]album(nil), s.albums...)
}

func (s *stubAlbumStore) get(id string) (album, bool) {
	for _, candidate := range s.albums {
		if candidate.ID == id {
			return candidate, true
		}
	}
	return album{}, false
}

func (s *stubAlbumStore) create(newAlbum album) {
	s.albums = append(s.albums, newAlbum)
}

func (s *stubAlbumStore) update(id string, updatedAlbum album) (album, bool) {
	for index, candidate := range s.albums {
		if candidate.ID == id {
			s.albums[index] = updatedAlbum
			return updatedAlbum, true
		}
	}
	return album{}, false
}

func (s *stubAlbumStore) delete(id string) bool {
	for index, candidate := range s.albums {
		if candidate.ID == id {
			s.albums = append(s.albums[:index], s.albums[index+1:]...)
			return true
		}
	}
	return false
}

func TestRouterAcceptsAlbumStoreImplementation(t *testing.T) {
	want := album{ID: "custom", Title: "Custom Store", Artist: "Test Artist", Price: 1}
	router := setupRouterWithStore(&stubAlbumStore{albums: []album{want}})
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/albums/custom", nil)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	var got album
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got != want {
		t.Fatalf("expected album %#v, got %#v", want, got)
	}
}

func TestGetAlbums(t *testing.T) {
	router := testRouter(t)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/albums", nil)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	var got []album
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	want := seedAlbums()
	if len(got) != len(want) {
		t.Fatalf("expected %d albums, got %d", len(want), len(got))
	}
}

func TestGetAlbumByID(t *testing.T) {
	router := testRouter(t)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/albums/2", nil)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	var got album
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	want := seedAlbums()[1]
	if got != want {
		t.Fatalf("expected album %#v, got %#v", want, got)
	}
}

func TestGetAlbumByIDNotFound(t *testing.T) {
	router := testRouter(t)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/albums/missing", nil)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, response.Code)
	}

	var got map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got["message"] != "album not found" {
		t.Fatalf("expected not-found message, got %#v", got)
	}
}

func TestPostAlbums(t *testing.T) {
	store := newAlbumStore(seedAlbums())
	router := setupRouterWithStore(store)
	response := httptest.NewRecorder()
	body := []byte(`{"id":"4","title":"Kind of Blue","artist":"Miles Davis","price":29.99}`)
	request := httptest.NewRequest(http.MethodPost, "/albums", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, response.Code)
	}
	if got := len(store.list()); got != 4 {
		t.Fatalf("expected 4 albums after creation, got %d", got)
	}
}

func TestPostAlbumsRejectsMalformedJSON(t *testing.T) {
	store := newAlbumStore(seedAlbums())
	before := len(store.list())
	router := setupRouterWithStore(store)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/albums", bytes.NewBufferString(`{"id":`))
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
	if got := len(store.list()); got != before {
		t.Fatalf("malformed request changed album count from %d to %d", before, got)
	}

	var got errorResponse
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Message != "invalid request body" {
		t.Fatalf("expected invalid-body message, got %q", got.Message)
	}
}

func TestPostAlbumsValidatesRequest(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		wantMessage string
	}{
		{name: "missing id", body: `{"title":"Kind of Blue","artist":"Miles Davis","price":29.99}`, wantMessage: "id is required"},
		{name: "blank title", body: `{"id":"4","title":" ","artist":"Miles Davis","price":29.99}`, wantMessage: "title is required"},
		{name: "missing artist", body: `{"id":"4","title":"Kind of Blue","price":29.99}`, wantMessage: "artist is required"},
		{name: "zero price", body: `{"id":"4","title":"Kind of Blue","artist":"Miles Davis","price":0}`, wantMessage: "price must be greater than zero"},
		{name: "negative price", body: `{"id":"4","title":"Kind of Blue","artist":"Miles Davis","price":-1}`, wantMessage: "price must be greater than zero"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newAlbumStore(seedAlbums())
			before := len(store.list())
			router := setupRouterWithStore(store)
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/albums", bytes.NewBufferString(tt.body))
			request.Header.Set("Content-Type", "application/json")

			router.ServeHTTP(response, request)

			if response.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
			}
			if got := len(store.list()); got != before {
				t.Fatalf("invalid request changed album count from %d to %d", before, got)
			}

			var got errorResponse
			if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if got.Message != tt.wantMessage {
				t.Fatalf("expected message %q, got %q", tt.wantMessage, got.Message)
			}
		})
	}
}

func TestPutAlbumByID(t *testing.T) {
	store := newAlbumStore(seedAlbums())
	router := setupRouterWithStore(store)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPut,
		"/albums/2",
		bytes.NewBufferString(`{"id":"2","title":"Night Lights","artist":"Gerry Mulligan","price":24.99}`),
	)
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	want := album{ID: "2", Title: "Night Lights", Artist: "Gerry Mulligan", Price: 24.99}
	got, ok := store.get("2")
	if !ok || got != want {
		t.Fatalf("expected stored album %#v, got %#v (found: %t)", want, got, ok)
	}
}

func TestPutAlbumByIDRejectsInvalidRequests(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		body        string
		wantStatus  int
		wantMessage string
	}{
		{name: "malformed body", path: "/albums/2", body: `{"id":`, wantStatus: http.StatusBadRequest, wantMessage: "invalid request body"},
		{name: "invalid album", path: "/albums/2", body: `{"id":"2","title":"","artist":"Gerry Mulligan","price":24.99}`, wantStatus: http.StatusBadRequest, wantMessage: "title is required"},
		{name: "mismatched id", path: "/albums/2", body: `{"id":"3","title":"Night Lights","artist":"Gerry Mulligan","price":24.99}`, wantStatus: http.StatusBadRequest, wantMessage: "album id must match path id"},
		{name: "missing album", path: "/albums/missing", body: `{"id":"missing","title":"Night Lights","artist":"Gerry Mulligan","price":24.99}`, wantStatus: http.StatusNotFound, wantMessage: "album not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newAlbumStore(seedAlbums())
			before := store.list()
			router := setupRouterWithStore(store)
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPut, tt.path, bytes.NewBufferString(tt.body))
			request.Header.Set("Content-Type", "application/json")

			router.ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, response.Code)
			}
			var got errorResponse
			if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if got.Message != tt.wantMessage {
				t.Fatalf("expected message %q, got %q", tt.wantMessage, got.Message)
			}
			if after := store.list(); fmt.Sprint(after) != fmt.Sprint(before) {
				t.Fatalf("failed update changed albums from %#v to %#v", before, after)
			}
		})
	}
}

func TestDeleteAlbumByID(t *testing.T) {
	store := newAlbumStore(seedAlbums())
	router := setupRouterWithStore(store)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodDelete, "/albums/2", nil)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, response.Code)
	}
	if response.Body.Len() != 0 {
		t.Fatalf("expected an empty response body, got %q", response.Body.String())
	}
	if _, ok := store.get("2"); ok {
		t.Fatal("expected album 2 to be deleted")
	}
}

func TestDeleteAlbumByIDNotFound(t *testing.T) {
	store := newAlbumStore(seedAlbums())
	before := len(store.list())
	router := setupRouterWithStore(store)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodDelete, "/albums/missing", nil)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, response.Code)
	}
	if got := len(store.list()); got != before {
		t.Fatalf("missing delete changed album count from %d to %d", before, got)
	}
}

func TestAlbumRoutesHandleConcurrentRequests(t *testing.T) {
	store := newAlbumStore(seedAlbums())
	router := setupRouterWithStore(store)
	before := len(store.list())

	const requestCount = 50
	errs := make(chan error, requestCount*2)
	var wg sync.WaitGroup

	for i := 0; i < requestCount; i++ {
		wg.Add(2)

		go func(i int) {
			defer wg.Done()
			body := fmt.Sprintf(
				`{"id":"concurrent-%d","title":"Album %d","artist":"Artist","price":1}`,
				i,
				i,
			)
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/albums", bytes.NewBufferString(body))
			request.Header.Set("Content-Type", "application/json")

			router.ServeHTTP(response, request)
			if response.Code != http.StatusCreated {
				errs <- fmt.Errorf("POST %d: expected status %d, got %d", i, http.StatusCreated, response.Code)
			}
		}(i)

		go func(i int) {
			defer wg.Done()
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/albums", nil)

			router.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				errs <- fmt.Errorf("GET %d: expected status %d, got %d", i, http.StatusOK, response.Code)
			}
		}(i)
	}

	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}

	if got, want := len(store.list()), before+requestCount; got != want {
		t.Fatalf("expected %d albums after concurrent creation, got %d", want, got)
	}
}

func TestRoutersUseIndependentStores(t *testing.T) {
	firstStore := newAlbumStore(seedAlbums())
	firstRouter := setupRouterWithStore(firstStore)
	secondStore := newAlbumStore(seedAlbums())
	secondRouter := setupRouterWithStore(secondStore)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/albums",
		bytes.NewBufferString(`{"id":"4","title":"Kind of Blue","artist":"Miles Davis","price":29.99}`),
	)
	request.Header.Set("Content-Type", "application/json")
	firstRouter.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, response.Code)
	}
	if got := len(firstStore.list()); got != 4 {
		t.Fatalf("expected first store to contain 4 albums, got %d", got)
	}

	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/albums", nil)
	secondRouter.ServeHTTP(response, request)

	var got []album
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if want := len(seedAlbums()); len(got) != want {
		t.Fatalf("expected second store to remain at %d albums, got %d", want, len(got))
	}
}
