package main

import (
	_ "embed"
	"net/http"

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

// @host localhost:8080
// @BasePath /

func setupRouter() *gin.Engine {
	return setupRouterWithStore(newAlbumStore(seedAlbums()))
}

func setupRouterWithStore(store albumStore) *gin.Engine {
	gin.EnableJsonDecoderDisallowUnknownFields()

	router := gin.Default()
	router.UseRawPath = true
	router.HandleMethodNotAllowed = true
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, errorResponse{Message: "route not found"})
	})
	router.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, errorResponse{Message: "method not allowed"})
	})
	handler := albumHandler{store: store}

	router.GET("/health", getHealth)
	router.GET("/albums", handler.getAlbums)
	router.GET("/albums/:id", handler.getAlbumByID)
	router.POST("/albums", handler.postAlbums)
	router.PUT("/albums/:id", handler.putAlbumByID)
	router.PATCH("/albums/:id", handler.patchAlbumByID)
	router.DELETE("/albums/:id", handler.deleteAlbumByID)

	// Swagger JSON endpoint (used by Scalar)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Scalar docs UI
	router.GET("/docs", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", scalarHTML)
	})

	return router
}

// getHealth reports whether the service process is ready to handle requests.
// @Summary Check service health
// @Description Report whether the service process is ready to handle requests
// @Tags health
// @Produce json
// @Success 200 {object} healthResponse
// @Router /health [get]
func getHealth(c *gin.Context) {
	c.JSON(http.StatusOK, healthResponse{Status: "ok"})
}
