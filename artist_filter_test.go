package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"testing"
)

func TestAlbumsArtistFilter(t *testing.T) {
	albums := []album{
		{ID: "1", Title: "First", Artist: "A + B", Price: 10},
		{ID: "2", Title: "Other", Artist: "Other", Price: 20},
		{ID: "3", Title: "Second", Artist: "A + B", Price: 30},
	}
	for _, scenario := range []struct {
		query string
		want  []album
		total string
	}{
		{query: "artist=A+%2B+B", want: []album{albums[0], albums[2]}, total: "2"},
		{query: "artist=A+%2B+B&limit=1&offset=1", want: albums[2:], total: "2"},
		{query: "artist=A+%2B+B&offset=1", want: albums[2:], total: "2"},
		{query: "artist=A+%2B+B&offset=2", total: "2"},
		{query: "artist=A+%2B+B&limit=" + strconv.Itoa(int(^uint(0)>>1)), want: []album{albums[0], albums[2]}, total: "2"},
		{query: "artist=a+%2B+b", total: "0"},
		{query: "artist=+A+%2B+B", total: "0"},
		{query: "artist=Missing&limit=1", total: "0"},
	} {
		t.Run(scenario.query, func(t *testing.T) {
			store := newAlbumStore(albums)
			router := setupRouterWithStore(store)
			for _, method := range []string{http.MethodGet, http.MethodHead} {
				response := httptest.NewRecorder()
				router.ServeHTTP(response, httptest.NewRequest(method, "/albums?"+scenario.query, nil))
				if response.Code != http.StatusOK || response.Header().Get("X-Total-Count") != scenario.total {
					t.Fatalf("%s: expected 200 with total %s, got %d %v", method, scenario.total, response.Code, response.Header())
				}
				if method == http.MethodHead {
					if response.Body.Len() != 0 {
						t.Fatalf("HEAD returned body: %s", response.Body)
					}
					continue
				}
				var got []album
				if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
					t.Fatal(err)
				}
				if got == nil || !slices.Equal(got, scenario.want) {
					t.Fatalf("expected array %v, got %s", scenario.want, response.Body)
				}
			}
			if !slices.Equal(store.list(), albums) {
				t.Fatal("filter changed stored albums")
			}
		})
	}
}

func TestAlbumsRejectInvalidArtistFilter(t *testing.T) {
	for _, query := range []string{"artist=", "artist=+%09", "artist=A&artist=B", "artist=%ff", "artist=%zz"} {
		for _, method := range []string{http.MethodGet, http.MethodHead} {
			t.Run(method+"/"+query, func(t *testing.T) {
				response := httptest.NewRecorder()
				setupRouter().ServeHTTP(response, httptest.NewRequest(method, "/albums?"+query, nil))
				if response.Code != http.StatusBadRequest || response.Header().Get("X-Total-Count") != "" {
					t.Fatalf("expected 400 without total, got %d %v", response.Code, response.Header())
				}
				if method == http.MethodHead && response.Body.Len() != 0 {
					t.Fatalf("HEAD returned body: %s", response.Body)
				}
			})
		}
	}
}
