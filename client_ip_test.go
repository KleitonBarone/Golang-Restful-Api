package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRouterUsesPeerClientIP(t *testing.T) {
	for _, peer := range []struct {
		address string
		ip      string
	}{
		{address: "192.0.2.10:1234", ip: "192.0.2.10"},
		{address: "[2001:db8::10]:1234", ip: "2001:db8::10"},
	} {
		for _, header := range []string{"", "X-Forwarded-For", "X-Real-IP"} {
			t.Run(peer.address+"/"+header, func(t *testing.T) {
				router := testRouter(t)
				var clientIP string
				router.Use(func(c *gin.Context) {
					clientIP = c.ClientIP()
				})
				router.GET("/client-ip", func(c *gin.Context) {
					c.Status(http.StatusNoContent)
				})
				request := httptest.NewRequest(http.MethodGet, "/client-ip", nil)
				request.RemoteAddr = peer.address
				if header != "" {
					request.Header.Set(header, "198.51.100.99")
				}
				response := httptest.NewRecorder()
				router.ServeHTTP(response, request)
				if response.Code != http.StatusNoContent {
					t.Fatalf("expected status 204, got %d", response.Code)
				}
				if clientIP != peer.ip {
					t.Fatalf("expected peer IP %q, got %q", peer.ip, clientIP)
				}
			})
		}
	}
}
