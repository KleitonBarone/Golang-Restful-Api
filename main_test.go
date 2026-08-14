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
