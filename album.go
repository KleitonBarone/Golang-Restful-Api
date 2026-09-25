package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

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

// UnmarshalJSON rejects null fields, which would otherwise decode like omitted fields.
func (p *albumPatch) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return errors.New("patch cannot be null")
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	for name, value := range fields {
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			switch strings.ToLower(name) {
			case "title", "artist", "price":
				return fmt.Errorf("%s cannot be null", name)
			}
		}
	}

	// A custom unmarshaler bypasses the outer decoder's unknown-field check.
	type patchFields albumPatch
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	return decoder.Decode((*patchFields)(p))
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
