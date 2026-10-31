package main

import (
	_ "embed"
	"net/http"
	"net/url"
	"strings"

	_ "project/web-service/docs"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

//go:embed docs/scalar.html
var scalarHTML []byte

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
	router.HEAD("/health", getHealth)
	router.Match([]string{http.MethodGet, http.MethodHead}, "/health/", redirectTrailingSlash)
	router.GET("/albums", handler.getAlbums)
	router.HEAD("/albums", omitResponseBody, handler.getAlbums)
	router.GET("/albums/:id", handler.getAlbumByID)
	router.HEAD("/albums/:id", omitResponseBody, handler.getAlbumByID)
	router.POST("/albums", handler.postAlbums)
	router.Match([]string{http.MethodGet, http.MethodHead, http.MethodPost}, "/albums/", redirectTrailingSlash)
	router.PUT("/albums/:id", handler.putAlbumByID)
	router.PATCH("/albums/:id", handler.patchAlbumByID)
	router.DELETE("/albums/:id", handler.deleteAlbumByID)
	router.Match([]string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodDelete},
		"/albums/:id/", redirectTrailingSlash)

	// Swagger JSON endpoint (used by Scalar)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Scalar docs UI
	serveDocs := func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", scalarHTML)
	}
	router.GET("/docs", serveDocs)
	router.HEAD("/docs", omitResponseBody, serveDocs)
	router.Match([]string{http.MethodGet, http.MethodHead}, "/docs/", redirectTrailingSlash)

	return router
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
// @Router /health [head]
func getHealth(c *gin.Context) {
	if c.Request.Method == http.MethodHead {
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.Status(http.StatusOK)
		return
	}
	c.JSON(http.StatusOK, healthResponse{Status: "ok"})
}
