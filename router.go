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

// @title Albums API
// @version 1.0
// @description A simple RESTful API for managing albums

// @host localhost:8080
// @BasePath /

func setupRouter() *gin.Engine {
	return setupRouterWithStore(newAlbumStore(seedAlbums()))
}

func setupRouterWithStore(store *albumStore) *gin.Engine {
	router := gin.Default()
	handler := albumHandler{store: store}

	router.GET("/albums", handler.getAlbums)
	router.GET("/albums/:id", handler.getAlbumByID)
	router.POST("/albums", handler.postAlbums)
	router.PUT("/albums/:id", handler.putAlbumByID)
	router.DELETE("/albums/:id", handler.deleteAlbumByID)

	// Swagger JSON endpoint (used by Scalar)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Scalar docs UI
	router.GET("/docs", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", scalarHTML)
	})

	return router
}
