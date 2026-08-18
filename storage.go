package main

import "sync"

type albumStore interface {
	list() []album
	get(id string) (album, bool)
	create(album)
	update(id string, album album) (album, bool)
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

func (s *inMemoryAlbumStore) create(newAlbum album) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.albums = append(s.albums, newAlbum)
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

func (s *inMemoryAlbumStore) delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for index, candidate := range s.albums {
		if candidate.ID == id {
			s.albums = append(s.albums[:index], s.albums[index+1:]...)
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
