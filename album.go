package main

import "strings"

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
