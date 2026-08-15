package main

import "sync"

type albumStore struct {
	mu     sync.RWMutex
	albums []album
}

func newAlbumStore(albums []album) *albumStore {
	return &albumStore{albums: append([]album(nil), albums...)}
}

func (s *albumStore) list() []album {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]album(nil), s.albums...)
}

func (s *albumStore) get(id string) (album, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, candidate := range s.albums {
		if candidate.ID == id {
			return candidate, true
		}
	}
	return album{}, false
}

func (s *albumStore) create(newAlbum album) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.albums = append(s.albums, newAlbum)
}

func (s *albumStore) update(id string, updatedAlbum album) (album, bool) {
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

func (s *albumStore) delete(id string) bool {
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
