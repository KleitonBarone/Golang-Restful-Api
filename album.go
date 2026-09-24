package main

import "strings"

// album represents data about a record album.
type album struct {
	ID     string  `json:"id"`
	Title  string  `json:"title"`
	Artist string  `json:"artist"`
	Price  float64 `json:"price"`
}

// albumPatch contains the mutable album fields supplied by a PATCH request.
type albumPatch struct {
	Title  *string  `json:"title,omitempty"`
	Artist *string  `json:"artist,omitempty"`
	Price  *float64 `json:"price,omitempty"`
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

func validateAlbumPatch(candidate albumPatch) string {
	switch {
	case candidate.Title != nil && strings.TrimSpace(*candidate.Title) == "":
		return "title is required"
	case candidate.Artist != nil && strings.TrimSpace(*candidate.Artist) == "":
		return "artist is required"
	case candidate.Price != nil && *candidate.Price <= 0:
		return "price must be greater than zero"
	default:
		return ""
	}
}

func (p albumPatch) apply(current album) album {
	if p.Title != nil {
		current.Title = *p.Title
	}
	if p.Artist != nil {
		current.Artist = *p.Artist
	}
	if p.Price != nil {
		current.Price = *p.Price
	}
	return current
}
