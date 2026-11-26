package main

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

//go:embed docs/scalar.html
var scalarHTML []byte

//go:embed docs/swagger.json
var openAPIJSON []byte

type healthResponse struct {
	Status string `json:"status"`
}

// @title Albums API
// @version 1.0
// @description A simple RESTful API for managing albums

// @BasePath /

func setupRouter() *gin.Engine {
	return setupRouterWithStore(newAlbumStore(seedAlbums()))
}

func setupRouterWithStore(store albumStore) *gin.Engine {
	router := gin.Default()
	// The service has no trusted-proxy configuration, so use the connection peer.
	router.ForwardedByClientIP = false
	router.UseRawPath = true
	router.UnescapePathValues = false
	router.Use(unescapePathParams)
	router.HandleMethodNotAllowed = true
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, errorResponse{Message: "route not found"})
	})
	router.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, errorResponse{Message: "method not allowed"})
	})
	handler := albumHandler{store: store}

	router.GET("/health", getHealth)

	// @Summary Check service health
	// @Description Return the service readiness status without a response body.
	// @Tags health
	// @Success 200 "Service ready"
	// @Router /health [head]
	router.HEAD("/health", getHealth)
	router.Match([]string{http.MethodGet, http.MethodHead}, "/health/", redirectTrailingSlash)
	router.GET("/albums", handler.getAlbums)

	// @Summary Check album collection metadata
	// @Description Return collection metadata without a response body. Artist, limit, and offset can each be specified once. Artist must be non-blank valid UTF-8; spaces are preserved when matching.
	// @Tags albums
	// @Param artist query string false "Exact, case-sensitive artist name; non-blank valid UTF-8"
	// @Param limit query int false "Maximum number of albums to return" minimum(1)
	// @Param offset query int false "Number of albums to skip" minimum(0)
	// @Success 200 "Collection metadata"
	// @Header 200 {integer} X-Total-Count "Number of matching albums before pagination, included when filtering or pagination is requested"
	// @Header 200 {string} Link "Relative next-page URL with rel=next, included when a limited page has more matching albums"
	// @Failure 400 "Invalid query"
	// @Router /albums [head]
	router.HEAD("/albums", omitResponseBody, handler.getAlbums)
	router.GET("/albums/:id", handler.getAlbumByID)

	// @Summary Check whether an album exists
	// @Description Return the album lookup status without a response body.
	// @Tags albums
	// @Param id path string true "Album ID"
	// @Success 200 "Album found"
	// @Failure 404 "Album not found"
	// @Router /albums/{id} [head]
	router.HEAD("/albums/:id", omitResponseBody, handler.getAlbumByID)
	router.POST("/albums", handler.postAlbums)
	router.Match([]string{http.MethodGet, http.MethodHead, http.MethodPost}, "/albums/", redirectTrailingSlash)
	router.PUT("/albums/:id", handler.putAlbumByID)
	router.PATCH("/albums/:id", handler.patchAlbumByID)
	router.DELETE("/albums/:id", handler.deleteAlbumByID)
	router.Match([]string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodDelete},
		"/albums/:id/", redirectTrailingSlash)

	// Scalar loads only the generated document; no Swagger UI assets are needed.
	registerEmbeddedDocument(router, "/swagger/doc.json", "doc.json", "application/json; charset=utf-8", openAPIJSON)
	registerEmbeddedDocument(router, "/docs", "scalar.html", "text/html; charset=utf-8", scalarHTML)

	return router
}

func registerEmbeddedDocument(router *gin.Engine, route, name, contentType string, content []byte) {
	// The content hash stays stable across restarts. ServeContent handles HTTP preconditions.
	etag := fmt.Sprintf(`"%x"`, sha256.Sum256(content))
	methods := []string{http.MethodGet, http.MethodHead}
	router.Match(methods, route, func(c *gin.Context) {
		c.Header("Content-Type", contentType)
		c.Header("ETag", etag)
		http.ServeContent(c.Writer, c.Request, name, time.Time{}, bytes.NewReader(content))
	})
	router.Match(methods, route+"/", redirectTrailingSlash)
}

// omitResponseBody lets HEAD reuse GET handlers, including their error responses.
func omitResponseBody(c *gin.Context) {
	c.Writer = headResponseWriter{c.Writer}
}

type headResponseWriter struct {
	gin.ResponseWriter
}

func (w headResponseWriter) Write(body []byte) (int, error) {
	w.WriteHeaderNow()
	return len(body), nil
}

func (w headResponseWriter) WriteString(body string) (int, error) {
	w.WriteHeaderNow()
	return len(body), nil
}

// redirectTrailingSlash preserves escaped paths and query strings without
// letting Gin rebuild the redirect or incorporate untrusted proxy headers.
func redirectTrailingSlash(c *gin.Context) {
	target := *c.Request.URL
	target.Path = strings.TrimSuffix(target.Path, "/")
	target.RawPath = strings.TrimSuffix(target.RawPath, "/")
	code := http.StatusTemporaryRedirect
	if c.Request.Method == http.MethodGet {
		code = http.StatusMovedPermanently
	}
	c.Redirect(code, target.RequestURI())
}

// unescapePathParams decodes raw path parameters once using path semantics.
// Gin's query decoding would turn literal plus signs into spaces.
func unescapePathParams(c *gin.Context) {
	if c.Request.URL.RawPath == "" {
		return // Gin already used the decoded URL.Path.
	}
	for i := range c.Params {
		value, err := url.PathUnescape(c.Params[i].Value)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, errorResponse{Message: "invalid request path"})
			return
		}
		c.Params[i].Value = value
	}
}

// getHealth reports whether the service process is ready to handle requests.
// @Summary Check service health
// @Description Report whether the service process is ready to handle requests
// @Tags health
// @Produce json
// @Success 200 {object} healthResponse
// @Router /health [get]
func getHealth(c *gin.Context) {
	if c.Request.Method == http.MethodHead {
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.Status(http.StatusOK)
		return
	}
	c.JSON(http.StatusOK, healthResponse{Status: "ok"})
}
