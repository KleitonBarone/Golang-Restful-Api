package main

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

const maxAlbumRequestBodyBytes int64 = 64 << 10

type albumHandler struct {
	store albumStore
}

// getAlbums responds with the list of all albums as JSON.
// @Summary List all albums
// @Description Get all albums in the collection
// @Tags albums
// @Produce json
// @Param limit query int false "Maximum number of albums to return" minimum(1)
// @Param offset query int false "Number of albums to skip" minimum(0)
// @Success 200 {array} album
// @Failure 400 {object} errorResponse
// @Router /albums [get]
func (h albumHandler) getAlbums(c *gin.Context) {
	limit, hasLimit, ok := positiveQueryInt(c, "limit")
	if !ok {
		return
	}
	offset, hasOffset, ok := nonNegativeQueryInt(c, "offset")
	if !ok {
		return
	}

	albums := h.store.list()
	if !hasLimit && !hasOffset {
		c.IndentedJSON(http.StatusOK, albums)
		return
	}
	if offset >= len(albums) {
		c.IndentedJSON(http.StatusOK, albums[len(albums):])
		return
	}

	albums = albums[offset:]
	if hasLimit && limit < len(albums) {
		albums = albums[:limit]
	}
	c.IndentedJSON(http.StatusOK, albums)
}

func positiveQueryInt(c *gin.Context, name string) (int, bool, bool) {
	raw, exists := c.GetQuery(name)
	if !exists {
		return 0, false, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		c.IndentedJSON(http.StatusBadRequest, errorResponse{Message: name + " must be a positive integer"})
		return 0, true, false
	}
	return value, true, true
}

func nonNegativeQueryInt(c *gin.Context, name string) (int, bool, bool) {
	raw, exists := c.GetQuery(name)
	if !exists {
		return 0, false, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		c.IndentedJSON(http.StatusBadRequest, errorResponse{Message: name + " must be a non-negative integer"})
		return 0, true, false
	}
	return value, true, true
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
// @Failure 409 {object} errorResponse
// @Failure 413 {object} errorResponse
// @Failure 415 {object} errorResponse
// @Router /albums [post]
func (h albumHandler) postAlbums(c *gin.Context) {
	newAlbum, ok := decodeAlbumRequest(c)
	if !ok {
		return
	}
	if validationError := validateAlbum(newAlbum); validationError != "" {
		c.IndentedJSON(http.StatusBadRequest, errorResponse{Message: validationError})
		return
	}

	if !h.store.create(newAlbum) {
		c.IndentedJSON(http.StatusConflict, errorResponse{Message: "album id already exists"})
		return
	}
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

// putAlbumByID replaces the album whose ID matches the path parameter.
// @Summary Update an album
// @Description Replace an existing album while retaining its ID
// @Tags albums
// @Accept json
// @Produce json
// @Param id path string true "Album ID"
// @Param album body album true "Replacement album"
// @Success 200 {object} album
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 413 {object} errorResponse
// @Failure 415 {object} errorResponse
// @Router /albums/{id} [put]
func (h albumHandler) putAlbumByID(c *gin.Context) {
	updatedAlbum, ok := decodeAlbumRequest(c)
	if !ok {
		return
	}
	if validationError := validateAlbum(updatedAlbum); validationError != "" {
		c.IndentedJSON(http.StatusBadRequest, errorResponse{Message: validationError})
		return
	}
	if updatedAlbum.ID != c.Param("id") {
		c.IndentedJSON(http.StatusBadRequest, errorResponse{Message: "album id must match path id"})
		return
	}

	updatedAlbum, ok = h.store.update(c.Param("id"), updatedAlbum)
	if !ok {
		c.IndentedJSON(http.StatusNotFound, errorResponse{Message: "album not found"})
		return
	}

	c.IndentedJSON(http.StatusOK, updatedAlbum)
}

// decodeAlbumRequest validates the media type and applies the shared body limit before decoding JSON.
func decodeAlbumRequest(c *gin.Context) (album, bool) {
	mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || mediaType != "application/json" {
		c.IndentedJSON(http.StatusUnsupportedMediaType, errorResponse{Message: "content type must be application/json"})
		return album{}, false
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxAlbumRequestBodyBytes)

	var requestedAlbum album
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&requestedAlbum); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			c.IndentedJSON(http.StatusRequestEntityTooLarge, errorResponse{Message: "request body too large"})
			return album{}, false
		}

		c.IndentedJSON(http.StatusBadRequest, errorResponse{Message: "invalid request body"})
		return album{}, false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			c.IndentedJSON(http.StatusRequestEntityTooLarge, errorResponse{Message: "request body too large"})
			return album{}, false
		}

		c.IndentedJSON(http.StatusBadRequest, errorResponse{Message: "invalid request body"})
		return album{}, false
	}

	return requestedAlbum, true
}

// deleteAlbumByID removes the album whose ID matches the path parameter.
// @Summary Delete an album
// @Description Remove an album from the collection by its ID
// @Tags albums
// @Produce json
// @Param id path string true "Album ID"
// @Success 204
// @Failure 404 {object} errorResponse
// @Router /albums/{id} [delete]
func (h albumHandler) deleteAlbumByID(c *gin.Context) {
	if !h.store.delete(c.Param("id")) {
		c.IndentedJSON(http.StatusNotFound, errorResponse{Message: "album not found"})
		return
	}

	c.Status(http.StatusNoContent)
}
