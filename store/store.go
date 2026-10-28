// Package store keeps bookmarks in memory, optionally persisting them to a JSON file.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// ErrNotFound is returned when no bookmark has the requested ID.
var ErrNotFound = errors.New("bookmark not found")

// Bookmark is a saved link.
type Bookmark struct {
	ID        int       `json:"id"`
	URL       string    `json:"url"`
	Title     string    `json:"title"`
	Tags      []string  `json:"tags,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Store is safe for concurrent use.
type Store struct {
	mu     sync.RWMutex
	path   string
	nextID int
	items  map[int]Bookmark
}

type snapshot struct {
	NextID    int        `json:"next_id"`
	Bookmarks []Bookmark `json:"bookmarks"`
}

// New returns an empty, in-memory store.
func New() *Store {
	return &Store{nextID: 1, items: map[int]Bookmark{}}
}

// Open returns a store that persists to path, loading any bookmarks already saved there.
func Open(path string) (*Store, error) {
	s := New()
	s.path = path

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}

	var snap snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	s.nextID = snap.NextID
	for _, b := range snap.Bookmarks {
		s.items[b.ID] = b
	}
	return s, nil
}

// Create stores b with a new ID and returns it.
func (s *Store) Create(b Bookmark) (Bookmark, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	b.ID = s.nextID
	b.Tags = normaliseTags(b.Tags)
	b.CreatedAt = now
	b.UpdatedAt = now

	s.nextID++
	s.items[b.ID] = b
	return b, s.persist()
}

// Get returns the bookmark with the given ID.
func (s *Store) Get(id int) (Bookmark, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	b, ok := s.items[id]
	if !ok {
		return Bookmark{}, ErrNotFound
	}
	return b, nil
}

// Update replaces the URL, title and tags of an existing bookmark.
func (s *Store) Update(id int, b Bookmark) (Bookmark, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	old, ok := s.items[id]
	if !ok {
		return Bookmark{}, ErrNotFound
	}
	old.URL = b.URL
	old.Title = b.Title
	old.Tags = normaliseTags(b.Tags)
	old.UpdatedAt = time.Now().UTC()

	s.items[id] = old
	return old, s.persist()
}

// Delete removes the bookmark with the given ID.
func (s *Store) Delete(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.items[id]; !ok {
		return ErrNotFound
	}
	delete(s.items, id)
	s.persist()
	return nil
}

// List returns one page of bookmarks, oldest first. Pages start at 1.
func (s *Store) List(page, size int) []Bookmark {
	s.mu.RLock()
	defer s.mu.RUnlock()

	all := s.sorted()
	start := page * size
	if start >= len(all) {
		return []Bookmark{}
	}
	end := min(start+size, len(all))
	return all[start:end]
}

// ByTag returns every bookmark carrying tag, oldest first.
func (s *Store) ByTag(tag string) []Bookmark {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tag = strings.ToLower(strings.TrimSpace(tag))
	out := []Bookmark{}
	for _, b := range s.sorted() {
		for _, t := range b.Tags {
			if t == tag {
				out = append(out, b)
				break
			}
		}
	}
	return out
}

// Count returns the number of stored bookmarks.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.items)
}

// sorted returns all bookmarks ordered by ID. The caller must hold s.mu.
func (s *Store) sorted() []Bookmark {
	all := make([]Bookmark, 0, len(s.items))
	for _, b := range s.items {
		all = append(all, b)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].ID < all[j].ID })
	return all
}

// persist writes the store to its file, if it has one. The caller must hold s.mu.
func (s *Store) persist() error {
	if s.path == "" {
		return nil
	}

	data, err := json.MarshalIndent(snapshot{NextID: s.nextID, Bookmarks: s.sorted()}, "", "  ")
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".bookmarks-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), s.path)
}

// normaliseTags lower-cases and trims tags, dropping empty ones and duplicates.
func normaliseTags(tags []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, t := range tags {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}
