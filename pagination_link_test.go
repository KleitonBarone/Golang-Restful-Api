package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestAlbumNextPageLink(t *testing.T) {
	maxInt := strconv.Itoa(int(^uint(0) >> 1))
	for _, scenario := range []struct {
		query string
		link  string
	}{
		{query: "limit=1", link: `</albums?limit=1&offset=1>; rel="next"`},
		{query: "limit=1&offset=1", link: `</albums?limit=1&offset=2>; rel="next"`},
		{query: "limit=2", link: `</albums?limit=2&offset=2>; rel="next"`},
		{query: "limit=1&offset=2"},
		{query: "limit=3"},
		{query: "limit=1&offset=3"},
		{query: "limit=1&offset=" + maxInt},
		{query: "limit=" + maxInt + "&offset=1"},
		{query: "offset=1"},
		{query: "artist=Missing&limit=1"},
		{query: ""},
	} {
		t.Run(scenario.query, func(t *testing.T) {
			router := setupRouter()
			for _, method := range []string{http.MethodGet, http.MethodHead} {
				response := httptest.NewRecorder()
				request := httptest.NewRequest(method, "/albums?"+scenario.query, nil)
				request.Header.Set("X-Forwarded-Prefix", "https://untrusted.example/")
				router.ServeHTTP(response, request)
				if response.Code != http.StatusOK || response.Header().Get("Link") != scenario.link {
					t.Fatalf("%s: got status %d, link %q; want 200, %q", method, response.Code, response.Header().Get("Link"), scenario.link)
				}
				if method == http.MethodHead && response.Body.Len() != 0 {
					t.Fatalf("HEAD returned a body: %s", response.Body)
				}
			}
		})
	}
}

func TestFollowFilteredAlbumPageLinks(t *testing.T) {
	albums := []album{
		{ID: "1", Title: "First", Artist: `A + B > "C"`, Price: 10},
		{ID: "2", Title: "Other", Artist: "Other", Price: 20},
		{ID: "3", Title: "Second", Artist: `A + B > "C"`, Price: 30},
	}
	router := setupRouterWithStore(newAlbumStore(albums))
	path := "/albums?limit=1&artist=A%20%2B%20B%20%3E%20%22C%22"
	var got []album
	for page := 0; ; page++ {
		if page > len(albums) {
			t.Fatal("pagination links did not terminate")
		}
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK || response.Header().Get("X-Total-Count") != "2" {
			t.Fatalf("unexpected filtered page: %d %v", response.Code, response.Header())
		}
		var items []album
		if err := json.Unmarshal(response.Body.Bytes(), &items); err != nil {
			t.Fatal(err)
		}
		got = append(got, items...)
		link := response.Header().Get("Link")
		head := httptest.NewRecorder()
		router.ServeHTTP(head, httptest.NewRequest(http.MethodHead, path, nil))
		if head.Code != http.StatusOK || head.Header().Get("Link") != link || head.Header().Get("X-Total-Count") != "2" || head.Body.Len() != 0 {
			t.Fatalf("HEAD metadata differs: %d %v %s", head.Code, head.Header(), head.Body)
		}
		if link == "" {
			break
		}
		if !strings.HasPrefix(link, "</albums?") || !strings.HasSuffix(link, `>; rel="next"`) {
			t.Fatalf("invalid next-page link: %q", link)
		}
		path = strings.TrimSuffix(strings.TrimPrefix(link, "<"), `>; rel="next"`)
	}
	if !slices.Equal(got, []album{albums[0], albums[2]}) {
		t.Fatalf("following links lost or duplicated matching albums: %+v", got)
	}
}
