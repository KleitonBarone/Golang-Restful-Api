package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthAndDocsRedirectsIgnoreForwardedPrefix(t *testing.T) {
	router := testRouter(t)
	for _, route := range []struct {
		method string
		path   string
		status int
	}{
		{http.MethodGet, "/health", http.StatusMovedPermanently},
		{http.MethodHead, "/health", http.StatusTemporaryRedirect},
		{http.MethodGet, "/docs", http.StatusMovedPermanently},
	} {
		for _, prefix := range []string{"", "/untrusted"} {
			t.Run(route.method+route.path+"/prefix="+prefix, func(t *testing.T) {
				request := httptest.NewRequest(route.method, route.path+"/?probe=ready&label=a%2Bb", nil)
				request.Header.Set("X-Forwarded-Prefix", prefix)
				response := httptest.NewRecorder()
				router.ServeHTTP(response, request)
				if response.Code != route.status {
					t.Fatalf("expected redirect status %d, got %d", route.status, response.Code)
				}
				location := response.Header().Get("Location")
				if want := route.path + "?probe=ready&label=a%2Bb"; location != want {
					t.Fatalf("expected location %q, got %q", want, location)
				}
				if route.method == http.MethodHead && response.Body.Len() != 0 {
					t.Fatal("HEAD redirect returned a response body")
				}

				response = httptest.NewRecorder()
				router.ServeHTTP(response, httptest.NewRequest(route.method, location, nil))
				if response.Code != http.StatusOK {
					t.Fatalf("redirect target returned %d: %s", response.Code, response.Body)
				}
			})
		}
	}
}
