package main

import (
	"slices"
	"sync"
	"testing"
)

func TestAlbumStoreDeleteClearsUnusedStorage(t *testing.T) {
	albums := seedAlbums()
	for _, scenario := range []struct {
		name    string
		initial []album
		id      string
		want    []album
	}{
		{name: "first", initial: albums, id: "1", want: albums[1:]},
		{name: "middle", initial: albums, id: "2", want: []album{albums[0], albums[2]}},
		{name: "last", initial: albums, id: "3", want: albums[:2]},
		{name: "only", initial: albums[:1], id: "1"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			store := newAlbumStore(scenario.initial)
			backing := store.albums
			if !store.delete(scenario.id) {
				t.Fatal("expected album to be deleted")
			}
			if got := store.list(); !slices.Equal(got, scenario.want) {
				t.Fatalf("expected remaining albums %v, got %v", scenario.want, got)
			}
			for _, unused := range backing[len(store.albums):] {
				if unused != (album{}) {
					t.Fatalf("unused backing storage retains album data: %v", unused)
				}
			}
		})
	}
}

func TestAlbumStorePageBounds(t *testing.T) {
	albums := seedAlbums()
	maxInt := int(^uint(0) >> 1)
	for _, scenario := range []struct {
		name   string
		limit  int
		offset int
		want   []album
	}{
		{name: "unbounded", want: albums},
		{name: "middle page", limit: 1, offset: 1, want: albums[1:2]},
		{name: "remaining albums", offset: 1, want: albums[1:]},
		{name: "maximum limit", limit: maxInt, offset: 1, want: albums[1:]},
		{name: "at end", limit: 1, offset: len(albums)},
		{name: "maximum offset", limit: maxInt, offset: maxInt},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			page, total := newAlbumStore(albums).listPage(scenario.limit, scenario.offset)
			if total != len(albums) || !slices.Equal(page, scenario.want) {
				t.Fatalf("expected page %v and total %d, got %v and %d", scenario.want, len(albums), page, total)
			}
		})
	}
	if page, total := newAlbumStore(nil).listPage(1, 0); len(page) != 0 || total != 0 {
		t.Fatalf("expected empty page and zero total, got %v and %d", page, total)
	}
}

func TestAlbumStorePageIsDetached(t *testing.T) {
	store := newAlbumStore(seedAlbums())
	page, _ := store.listPage(1, 1)
	original := page[0]
	page[0].Title = "Changed by caller"
	if got, _ := store.get(original.ID); got != original {
		t.Fatalf("changing page altered stored album: %v", got)
	}
	updated := original
	updated.Artist = "Changed by store"
	store.update(original.ID, updated)
	if page[0].Artist != original.Artist {
		t.Fatal("updating store altered a previously returned page")
	}
}

func TestAlbumStorePageAndTotalShareSnapshot(t *testing.T) {
	store := newAlbumStore(nil)
	var writers sync.WaitGroup
	writers.Add(1)
	go func() {
		defer writers.Done()
		for range 1000 {
			store.create(album{ID: "temporary"})
			store.delete("temporary")
		}
	}()
	defer writers.Wait()
	for range 1000 {
		page, total := store.listPage(0, 0)
		if len(page) != total {
			t.Fatalf("page and total came from different snapshots: length %d, total %d", len(page), total)
		}
	}
}
