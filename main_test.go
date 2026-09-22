package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func testRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	return setupRouterWithStore(newAlbumStore(seedAlbums()))
}

func TestListenAddress(t *testing.T) {
	t.Run("uses the default when unset", func(t *testing.T) {
		t.Setenv("LISTEN_ADDRESS", "")

		if got, want := listenAddress(), "localhost:8080"; got != want {
			t.Fatalf("expected listen address %q, got %q", want, got)
		}
	})

	t.Run("uses the configured address", func(t *testing.T) {
		t.Setenv("LISTEN_ADDRESS", ":9090")

		if got, want := listenAddress(), ":9090"; got != want {
			t.Fatalf("expected listen address %q, got %q", want, got)
		}
	})
}

func TestNewHTTPServerBoundsConnectionWaits(t *testing.T) {
	server := newHTTPServer(http.NotFoundHandler())

	if got, want := server.ReadTimeout, 10*time.Second; got != want {
		t.Fatalf("expected read timeout %s, got %s", want, got)
	}
	if got, want := server.WriteTimeout, 10*time.Second; got != want {
		t.Fatalf("expected write timeout %s, got %s", want, got)
	}
	if got, want := server.ReadHeaderTimeout, 5*time.Second; got != want {
		t.Fatalf("expected read header timeout %s, got %s", want, got)
	}
	if got, want := server.MaxHeaderBytes, 16<<10; got != want {
		t.Fatalf("expected maximum header size %d bytes, got %d", want, got)
	}
	if got, want := server.IdleTimeout, 60*time.Second; got != want {
		t.Fatalf("expected idle timeout %s, got %s", want, got)
	}
}

func TestRunHTTPServerDrainsInFlightRequestAfterCancellation(t *testing.T) {
	requestStarted := make(chan struct{})
	releaseRequest := make(chan struct{})
	server := &http.Server{
		Handler: http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			close(requestStarted)
			<-releaseRequest
			_, _ = response.Write([]byte("done"))
		}),
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- runHTTPServer(ctx, server, listener, time.Second)
	}()

	responseDone := make(chan error, 1)
	go func() {
		response, err := http.Get("http://" + listener.Addr().String())
		if err != nil {
			responseDone <- err
			return
		}
		defer response.Body.Close()
		body, err := io.ReadAll(response.Body)
		if err == nil && string(body) != "done" {
			err = fmt.Errorf("expected response body %q, got %q", "done", body)
		}
		responseDone <- err
	}()

	select {
	case <-requestStarted:
	case <-time.After(time.Second):
		t.Fatal("request did not reach the server")
	}
	cancel()

	select {
	case err := <-serverDone:
		t.Fatalf("server returned before the in-flight request completed: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	close(releaseRequest)
	if err := <-responseDone; err != nil {
		t.Fatalf("request failed during graceful shutdown: %v", err)
	}
	if err := <-serverDone; err != nil {
		t.Fatalf("run server: %v", err)
	}
}

func TestRunHTTPServerForceClosesAfterShutdownTimeout(t *testing.T) {
	requestStarted := make(chan struct{})
	requestStopped := make(chan struct{})
	server := &http.Server{
		Handler: http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
			close(requestStarted)
			<-request.Context().Done()
			close(requestStopped)
		}),
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- runHTTPServer(ctx, server, listener, 50*time.Millisecond)
	}()

	requestDone := make(chan struct{})
	go func() {
		response, err := http.Get("http://" + listener.Addr().String())
		if err == nil {
			_ = response.Body.Close()
		}
		close(requestDone)
	}()

	select {
	case <-requestStarted:
	case <-time.After(time.Second):
		t.Fatal("request did not reach the server")
	}
	cancel()

	select {
	case err := <-serverDone:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected shutdown deadline error, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("server did not return after the shutdown deadline")
	}

	select {
	case <-requestStopped:
	case <-time.After(time.Second):
		t.Fatal("force close did not stop the in-flight request")
	}
	select {
	case <-requestDone:
	case <-time.After(time.Second):
		t.Fatal("force close did not release the client request")
	}
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

func (s *stubAlbumStore) create(newAlbum album) bool {
	for _, candidate := range s.albums {
		if candidate.ID == newAlbum.ID {
			return false
		}
	}
	s.albums = append(s.albums, newAlbum)
	return true
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

func TestHealth(t *testing.T) {
	router := testRouter(t)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health", nil)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	var got map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got["status"] != "ok" {
		t.Fatalf("expected healthy status, got %#v", got)
	}
}

func TestRouterReturnsJSONForRoutingErrors(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		path        string
		wantStatus  int
		wantMessage string
	}{
		{
			name:        "unmatched route",
			method:      http.MethodGet,
			path:        "/missing",
			wantStatus:  http.StatusNotFound,
			wantMessage: "route not found",
		},
		{
			name:        "unsupported method",
			method:      http.MethodPatch,
			path:        "/albums/1",
			wantStatus:  http.StatusMethodNotAllowed,
			wantMessage: "method not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(tt.method, tt.path, nil)

			testRouter(t).ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, response.Code)
			}
			if got := response.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
				t.Fatalf("expected JSON content type, got %q", got)
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

func TestGetAlbumsPagination(t *testing.T) {
	tests := []struct {
		name string
		path string
		want []album
	}{
		{name: "limit", path: "/albums?limit=2", want: seedAlbums()[:2]},
		{name: "offset", path: "/albums?offset=1", want: seedAlbums()[1:]},
		{name: "limit and offset", path: "/albums?limit=1&offset=1", want: seedAlbums()[1:2]},
		{name: "limit exceeds collection", path: "/albums?limit=10", want: seedAlbums()},
		{name: "offset reaches end", path: "/albums?offset=3", want: []album{}},
		{name: "offset exceeds collection", path: "/albums?offset=10", want: []album{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, tt.path, nil)

			testRouter(t).ServeHTTP(response, request)

			if response.Code != http.StatusOK {
				t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
			}
			var got []album
			if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if fmt.Sprint(got) != fmt.Sprint(tt.want) {
				t.Fatalf("expected albums %#v, got %#v", tt.want, got)
			}
		})
	}
}

func TestGetAlbumsRejectsInvalidPagination(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		wantMessage string
	}{
		{name: "empty limit", path: "/albums?limit=", wantMessage: "limit must be a positive integer"},
		{name: "zero limit", path: "/albums?limit=0", wantMessage: "limit must be a positive integer"},
		{name: "negative limit", path: "/albums?limit=-1", wantMessage: "limit must be a positive integer"},
		{name: "non-integer limit", path: "/albums?limit=one", wantMessage: "limit must be a positive integer"},
		{name: "empty offset", path: "/albums?offset=", wantMessage: "offset must be a non-negative integer"},
		{name: "negative offset", path: "/albums?offset=-1", wantMessage: "offset must be a non-negative integer"},
		{name: "non-integer offset", path: "/albums?offset=one", wantMessage: "offset must be a non-negative integer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, tt.path, nil)

			testRouter(t).ServeHTTP(response, request)

			if response.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
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
	if got, want := response.Header().Get("Location"), "/albums/4"; got != want {
		t.Fatalf("expected location %q, got %q", want, got)
	}
	if got := len(store.list()); got != 4 {
		t.Fatalf("expected 4 albums after creation, got %d", got)
	}
}

func TestAlbumMutationsRejectNonJSONContentType(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{
			name:   "create",
			method: http.MethodPost,
			path:   "/albums",
			body:   `{"id":"4","title":"Kind of Blue","artist":"Miles Davis","price":29.99}`,
		},
		{
			name:   "update",
			method: http.MethodPut,
			path:   "/albums/2",
			body:   `{"id":"2","title":"Night Lights","artist":"Gerry Mulligan","price":24.99}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newAlbumStore(seedAlbums())
			before := store.list()
			response := httptest.NewRecorder()
			request := httptest.NewRequest(tt.method, tt.path, bytes.NewBufferString(tt.body))
			request.Header.Set("Content-Type", "text/plain")

			setupRouterWithStore(store).ServeHTTP(response, request)

			if response.Code != http.StatusUnsupportedMediaType {
				t.Fatalf("expected status %d, got %d", http.StatusUnsupportedMediaType, response.Code)
			}
			if after := store.list(); fmt.Sprint(after) != fmt.Sprint(before) {
				t.Fatalf("unsupported media type changed albums from %#v to %#v", before, after)
			}

			var got errorResponse
			if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if got.Message != "content type must be application/json" {
				t.Fatalf("expected unsupported-media-type message, got %q", got.Message)
			}
		})
	}
}

func TestPostAlbumsRejectsDuplicateID(t *testing.T) {
	store := newAlbumStore(seedAlbums())
	before := store.list()
	router := setupRouterWithStore(store)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/albums",
		bytes.NewBufferString(`{"id":"2","title":"Duplicate","artist":"Another Artist","price":10}`),
	)
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, response.Code)
	}
	if after := store.list(); fmt.Sprint(after) != fmt.Sprint(before) {
		t.Fatalf("duplicate create changed albums from %#v to %#v", before, after)
	}

	var got errorResponse
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Message != "album id already exists" {
		t.Fatalf("expected duplicate-id message, got %q", got.Message)
	}
}

func TestPostAlbumsCreatesDuplicateIDOnlyOnceConcurrently(t *testing.T) {
	store := newAlbumStore(nil)
	router := setupRouterWithStore(store)

	const requestCount = 25
	var created atomic.Int32
	var conflicts atomic.Int32
	errs := make(chan error, requestCount)
	var wg sync.WaitGroup

	for range requestCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			response := httptest.NewRecorder()
			request := httptest.NewRequest(
				http.MethodPost,
				"/albums",
				bytes.NewBufferString(`{"id":"shared","title":"Concurrent","artist":"Test Artist","price":10}`),
			)
			request.Header.Set("Content-Type", "application/json")

			router.ServeHTTP(response, request)

			switch response.Code {
			case http.StatusCreated:
				created.Add(1)
			case http.StatusConflict:
				conflicts.Add(1)
			default:
				errs <- fmt.Errorf("expected status 201 or 409, got %d", response.Code)
			}
		}()
	}

	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
	if got := created.Load(); got != 1 {
		t.Fatalf("expected one created response, got %d", got)
	}
	if got := conflicts.Load(); got != requestCount-1 {
		t.Fatalf("expected %d conflict responses, got %d", requestCount-1, got)
	}
	if got := len(store.list()); got != 1 {
		t.Fatalf("expected one stored album, got %d", got)
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

func TestPostAlbumsRejectsOversizedBody(t *testing.T) {
	store := newAlbumStore(seedAlbums())
	before := store.list()
	response := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/albums",
		bytes.NewBufferString(fmt.Sprintf(`{"id":"4","title":"%s","artist":"Miles Davis","price":29.99}`, string(bytes.Repeat([]byte("x"), int(maxAlbumRequestBodyBytes))))),
	)
	request.Header.Set("Content-Type", "application/json")

	setupRouterWithStore(store).ServeHTTP(response, request)

	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status %d, got %d", http.StatusRequestEntityTooLarge, response.Code)
	}
	if after := store.list(); fmt.Sprint(after) != fmt.Sprint(before) {
		t.Fatalf("oversized request changed albums from %#v to %#v", before, after)
	}

	var got errorResponse
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Message != "request body too large" {
		t.Fatalf("expected oversized-body message, got %q", got.Message)
	}
}

func TestPostAlbumsRejectsOversizedTrailingData(t *testing.T) {
	store := newAlbumStore(seedAlbums())
	before := store.list()
	body := append(
		[]byte(`{"id":"4","title":"Kind of Blue","artist":"Miles Davis","price":29.99}`),
		bytes.Repeat([]byte(" "), int(maxAlbumRequestBodyBytes))...,
	)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/albums", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")

	setupRouterWithStore(store).ServeHTTP(response, request)

	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status %d, got %d", http.StatusRequestEntityTooLarge, response.Code)
	}
	if after := store.list(); fmt.Sprint(after) != fmt.Sprint(before) {
		t.Fatalf("oversized request changed albums from %#v to %#v", before, after)
	}

	var got errorResponse
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Message != "request body too large" {
		t.Fatalf("expected oversized-body message, got %q", got.Message)
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
		{name: "unknown field", body: `{"id":"4","title":"Kind of Blue","artist":"Miles Davis","price":29.99,"genre":"jazz"}`, wantMessage: "invalid request body"},
		{name: "trailing JSON value", body: `{"id":"4","title":"Kind of Blue","artist":"Miles Davis","price":29.99} {}`, wantMessage: "invalid request body"},
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
		{name: "unknown field", path: "/albums/2", body: `{"id":"2","title":"Night Lights","artist":"Gerry Mulligan","price":24.99,"genre":"jazz"}`, wantStatus: http.StatusBadRequest, wantMessage: "invalid request body"},
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

func TestPutAlbumByIDRejectsOversizedBody(t *testing.T) {
	store := newAlbumStore(seedAlbums())
	before := store.list()
	response := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPut,
		"/albums/2",
		bytes.NewBufferString(fmt.Sprintf(`{"id":"2","title":"%s","artist":"Gerry Mulligan","price":24.99}`, string(bytes.Repeat([]byte("x"), int(maxAlbumRequestBodyBytes))))),
	)
	request.Header.Set("Content-Type", "application/json")

	setupRouterWithStore(store).ServeHTTP(response, request)

	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status %d, got %d", http.StatusRequestEntityTooLarge, response.Code)
	}
	if after := store.list(); fmt.Sprint(after) != fmt.Sprint(before) {
		t.Fatalf("oversized request changed albums from %#v to %#v", before, after)
	}

	var got errorResponse
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Message != "request body too large" {
		t.Fatalf("expected oversized-body message, got %q", got.Message)
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
