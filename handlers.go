package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type albumHandler struct {
	store *albumStore
}

// getAlbums responds with the list of all albums as JSON.
// @Summary List all albums
// @Description Get all albums in the collection
// @Tags albums
// @Produce json
// @Success 200 {array} album
// @Router /albums [get]
func (h albumHandler) getAlbums(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, h.store.list())
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
func (h albumHandler) postAlbums(c *gin.Context) {
	var newAlbum album

	if err := c.ShouldBindJSON(&newAlbum); err != nil {
		c.IndentedJSON(http.StatusBadRequest, errorResponse{Message: "invalid request body"})
		return
	}
	if validationError := validateAlbum(newAlbum); validationError != "" {
		c.IndentedJSON(http.StatusBadRequest, errorResponse{Message: validationError})
		return
	}

	h.store.create(newAlbum)
	c.IndentedJSON(http.StatusCreated, newAlbum)
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
func (h albumHandler) getAlbumByID(c *gin.Context) {
	currentAlbum, ok := h.store.get(c.Param("id"))
	if !ok {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "album not found"})
		return
	}

	c.IndentedJSON(http.StatusOK, currentAlbum)
}
