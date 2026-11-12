package main

import (
	"slices"
	"sync"
)

type albumStore interface {
	list() []album
	// An empty artist disables filtering; otherwise the match is exact.
	count(artist string) int
	// listPage returns a detached filtered page and total from one snapshot. Zero limit is unbounded; offset must be non-negative.
	listPage(limit, offset int, artist string) ([]album, int)
	get(id string) (album, bool)
	create(album) bool
	update(id string, album album) (album, bool)
	patch(id string, changes albumPatch) (album, bool)
	delete(id string) bool
}

type inMemoryAlbumStore struct {
	mu     sync.RWMutex
	albums []album
}

var _ albumStore = (*inMemoryAlbumStore)(nil)

func newAlbumStore(albums []album) *inMemoryAlbumStore {
	return &inMemoryAlbumStore{albums: append([]album(nil), albums...)}
}

func (s *inMemoryAlbumStore) list() []album {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]album(nil), s.albums...)
}

func (s *inMemoryAlbumStore) count(artist string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if artist == "" {
		return len(s.albums)
	}
	total := 0
	for _, candidate := range s.albums {
		if candidate.Artist == artist {
			total++
		}
	}
	return total
}

func (s *inMemoryAlbumStore) listPage(limit, offset int, artist string) ([]album, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if artist != "" {
		var page []album
		total := 0
		for _, candidate := range s.albums {
			if candidate.Artist != artist {
				continue
			}
			if total >= offset && (limit == 0 || len(page) < limit) {
				page = append(page, candidate)
			}
			total++
		}
		return page, total
	}
	total := len(s.albums)
	if offset >= total {
		return nil, total
	}
	end := total
	if limit > 0 && limit < total-offset {
		end = offset + limit
	}
	return append([]album(nil), s.albums[offset:end]...), total
}

func (s *inMemoryAlbumStore) get(id string) (album, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, candidate := range s.albums {
		if candidate.ID == id {
			return candidate, true
		}
	}
	return album{}, false
}

func (s *inMemoryAlbumStore) create(newAlbum album) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, candidate := range s.albums {
		if candidate.ID == newAlbum.ID {
			return false
		}
	}
	s.albums = append(s.albums, newAlbum)
	return true
}

func (s *inMemoryAlbumStore) update(id string, updatedAlbum album) (album, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for index, candidate := range s.albums {
		if candidate.ID == id {
			s.albums[index] = updatedAlbum
			return updatedAlbum, true
		}
	}
	return album{}, false
}

func (s *inMemoryAlbumStore) patch(id string, changes albumPatch) (album, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for index, candidate := range s.albums {
		if candidate.ID == id {
			updatedAlbum := changes.apply(candidate)
			s.albums[index] = updatedAlbum
			return updatedAlbum, true
		}
	}
	return album{}, false
}

func (s *inMemoryAlbumStore) delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for index, candidate := range s.albums {
		if candidate.ID == id {
			s.albums = slices.Delete(s.albums, index, index+1)
			return true
		}
	}
	return false
}

func seedAlbums() []album {
	return []album{
		{ID: "1", Title: "Blue Train", Artist: "John Coltrane", Price: 56.99},
		{ID: "2", Title: "Jeru", Artist: "Gerry Mulligan", Price: 17.99},
		{ID: "3", Title: "Sarah Vaughan and Clifford Brown", Artist: "Sarah Vaughan", Price: 39.99},
	}
}
