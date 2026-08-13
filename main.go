package main

import (
	_ "embed"
	"net/http"
	"strings"
	"sync"

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

// album represents data about a record album.
type album struct {
	ID     string  `json:"id"`
	Title  string  `json:"title"`
	Artist string  `json:"artist"`
	Price  float64 `json:"price"`
}

type errorResponse struct {
	Message string `json:"message"`
}

// albums slice to seed record album data.
var albums = []album{
	{ID: "1", Title: "Blue Train", Artist: "John Coltrane", Price: 56.99},
	{ID: "2", Title: "Jeru", Artist: "Gerry Mulligan", Price: 17.99},
	{ID: "3", Title: "Sarah Vaughan and Clifford Brown", Artist: "Sarah Vaughan", Price: 39.99},
}
var albumsMu sync.RWMutex

func main() {
	router := setupRouter()
	router.Run("localhost:8080")
}

func setupRouter() *gin.Engine {
	router := gin.Default()
	router.GET("/albums", getAlbums)
	router.GET("/albums/:id", getAlbumByID)
	router.POST("/albums", postAlbums)

	// Swagger JSON endpoint (used by Scalar)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Scalar docs UI
	router.GET("/docs", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", scalarHTML)
	})

	return router
}

// getAlbums responds with the list of all albums as JSON.
// @Summary List all albums
// @Description Get all albums in the collection
// @Tags albums
// @Produce json
// @Success 200 {array} album
// @Router /albums [get]
func getAlbums(c *gin.Context) {
	albumsMu.RLock()
	currentAlbums := append([]album(nil), albums...)
	albumsMu.RUnlock()

	c.IndentedJSON(http.StatusOK, currentAlbums)
}

// postAlbums adds an album from JSON received in the request body.
// @Summary Create an album
// @Description Add a new album to the collection
// @Tags albums
// @Accept json
// @Produce json
// @Param album body album true "Album to create"
// @Success 201 {object} album
// @Failure 400 {object} errorResponse
// @Router /albums [post]
func postAlbums(c *gin.Context) {
	var newAlbum album

	if err := c.ShouldBindJSON(&newAlbum); err != nil {
		c.IndentedJSON(http.StatusBadRequest, errorResponse{Message: "invalid request body"})
		return
	}
	if validationError := validateAlbum(newAlbum); validationError != "" {
		c.IndentedJSON(http.StatusBadRequest, errorResponse{Message: validationError})
		return
	}

	albumsMu.Lock()
	albums = append(albums, newAlbum)
	albumsMu.Unlock()

	c.IndentedJSON(http.StatusCreated, newAlbum)
}

func validateAlbum(candidate album) string {
	switch {
	case strings.TrimSpace(candidate.ID) == "":
		return "id is required"
	case strings.TrimSpace(candidate.Title) == "":
		return "title is required"
	case strings.TrimSpace(candidate.Artist) == "":
		return "artist is required"
	case candidate.Price <= 0:
		return "price must be greater than zero"
	default:
		return ""
	}
}

// getAlbumByID locates the album whose ID value matches the id
// parameter sent by the client, then returns that album as a response.
// @Summary Get an album by ID
// @Description Get a single album by its ID
// @Tags albums
// @Produce json
// @Param id path string true "Album ID"
// @Success 200 {object} album
// @Failure 404 {object} map[string]string
// @Router /albums/{id} [get]
func getAlbumByID(c *gin.Context) {
	id := c.Param("id")

	albumsMu.RLock()
	for _, a := range albums {
		if a.ID == id {
			albumsMu.RUnlock()
			c.IndentedJSON(http.StatusOK, a)
			return
		}
	}
	albumsMu.RUnlock()

	c.IndentedJSON(http.StatusNotFound, gin.H{"message": "album not found"})
}
