package main

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// Compare small and near-request-limit albums to expose payload work in HEAD.
func BenchmarkHeadAlbumByID(b *testing.B) {
	for _, size := range []int{32, 60 << 10} {
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			store := newAlbumStore([]album{{ID: "1", Title: strings.Repeat("x", size), Artist: "Artist", Price: 10}})
			handler := albumHandler{store: store}
			router := gin.New()
			router.HEAD("/albums/:id", omitResponseBody, handler.getAlbumByID)
			request := httptest.NewRequest(http.MethodHead, "/albums/1", nil)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				response := httptest.NewRecorder()
				router.ServeHTTP(response, request)
				if response.Code != http.StatusOK || response.Body.Len() != 0 {
					b.Fatalf("expected bodyless 200, got %d %s", response.Code, response.Body)
				}
			}
		})
	}
}
