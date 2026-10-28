package store

import (
	"errors"
	"reflect"
	"testing"
)

func TestCreateAssignsIDs(t *testing.T) {
	s := New()

	a, err := s.Create(Bookmark{URL: "https://go.dev", Title: "Go"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Create(Bookmark{URL: "https://pkg.go.dev", Title: "Packages"})
	if err != nil {
		t.Fatal(err)
	}

	if a.ID != 1 || b.ID != 2 {
		t.Errorf("got IDs %d and %d, want 1 and 2", a.ID, b.ID)
	}
	if a.CreatedAt.IsZero() {
		t.Error("CreatedAt not set")
	}
}

func TestCreateNormalisesTags(t *testing.T) {
	s := New()

	b, err := s.Create(Bookmark{URL: "https://go.dev", Title: "Go", Tags: []string{" Go ", "go", "", "Docs"}})
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"go", "docs"}
	if !reflect.DeepEqual(b.Tags, want) {
		t.Errorf("got tags %q, want %q", b.Tags, want)
	}
}

func TestGet(t *testing.T) {
	s := New()
	created, _ := s.Create(Bookmark{URL: "https://go.dev", Title: "Go"})

	got, err := s.Get(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Go" {
		t.Errorf("got title %q, want %q", got.Title, "Go")
	}

	if _, err := s.Get(99); !errors.Is(err, ErrNotFound) {
		t.Errorf("got error %v, want ErrNotFound", err)
	}
}

func TestDelete(t *testing.T) {
	s := New()
	created, _ := s.Create(Bookmark{URL: "https://go.dev", Title: "Go"})

	if err := s.Delete(created.ID); err != nil {
		t.Fatal(err)
	}
	if s.Count() != 0 {
		t.Errorf("got %d bookmarks after delete, want 0", s.Count())
	}
	if err := s.Delete(created.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("second delete: got error %v, want ErrNotFound", err)
	}
}

func TestByTag(t *testing.T) {
	s := New()
	s.Create(Bookmark{URL: "https://go.dev", Title: "Go", Tags: []string{"go"}})
	s.Create(Bookmark{URL: "https://www.rust-lang.org", Title: "Rust", Tags: []string{"rust"}})
	s.Create(Bookmark{URL: "https://gobyexample.com", Title: "Go by Example", Tags: []string{"go", "learning"}})

	got := s.ByTag("Go")
	if len(got) != 2 || got[0].Title != "Go" || got[1].Title != "Go by Example" {
		t.Errorf("got %+v, want the two Go bookmarks in order", got)
	}
}
