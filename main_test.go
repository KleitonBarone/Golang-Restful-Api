package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func testRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	return setupRouter()
}

func preserveAlbums(t *testing.T) {
	t.Helper()
	original := append([]album(nil), albums...)
	t.Cleanup(func() {
		albums = original
	})
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
	if len(got) != len(albums) {
		t.Fatalf("expected %d albums, got %d", len(albums), len(got))
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
	if got != albums[1] {
		t.Fatalf("expected album %#v, got %#v", albums[1], got)
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
	preserveAlbums(t)
	router := testRouter(t)
	response := httptest.NewRecorder()
	body := []byte(`{"id":"4","title":"Kind of Blue","artist":"Miles Davis","price":29.99}`)
	request := httptest.NewRequest(http.MethodPost, "/albums", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, response.Code)
	}
	if len(albums) != 4 {
		t.Fatalf("expected 4 albums after creation, got %d", len(albums))
	}
}

func TestPostAlbumsRejectsMalformedJSON(t *testing.T) {
	preserveAlbums(t)
	before := len(albums)
	router := testRouter(t)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/albums", bytes.NewBufferString(`{"id":`))
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
	if len(albums) != before {
		t.Fatalf("malformed request changed album count from %d to %d", before, len(albums))
	}
}
