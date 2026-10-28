package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bookmarks/store"
)

func newServer(t *testing.T) (*httptest.Server, *store.Store) {
	t.Helper()
	s := store.New()
	srv := httptest.NewServer(New(s).Routes())
	t.Cleanup(srv.Close)
	return srv, s
}

func do(t *testing.T, method, url, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func TestHealth(t *testing.T) {
	srv, _ := newServer(t)

	resp := do(t, http.MethodGet, srv.URL+"/healthz", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got status %d, want 200", resp.StatusCode)
	}
}

func TestCreate(t *testing.T) {
	srv, _ := newServer(t)

	resp := do(t, http.MethodPost, srv.URL+"/bookmarks", `{"url":"https://go.dev","title":"Go","tags":["go"]}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("got status %d, want 201", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "/bookmarks/1" {
		t.Errorf("got Location %q, want /bookmarks/1", loc)
	}

	var b store.Bookmark
	if err := json.NewDecoder(resp.Body).Decode(&b); err != nil {
		t.Fatal(err)
	}
	if b.ID != 1 || b.Title != "Go" {
		t.Errorf("got %+v", b)
	}
}

func TestCreateRejectsInvalidInput(t *testing.T) {
	srv, _ := newServer(t)

	cases := map[string]string{
		"not JSON":       `nope`,
		"unknown field":  `{"url":"https://go.dev","title":"Go","colour":"blue"}`,
		"relative URL":   `{"url":"/docs","title":"Docs"}`,
		"ftp URL":        `{"url":"ftp://example.com","title":"FTP"}`,
		"missing title":  `{"url":"https://go.dev"}`,
		"blank title":    `{"url":"https://go.dev","title":"   "}`,
		"title too long": `{"url":"https://go.dev","title":"` + strings.Repeat("x", 201) + `"}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			resp := do(t, http.MethodPost, srv.URL+"/bookmarks", body)
			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("got status %d, want 400", resp.StatusCode)
			}
		})
	}
}

func TestGet(t *testing.T) {
	srv, s := newServer(t)
	s.Create(store.Bookmark{URL: "https://go.dev", Title: "Go"})

	if resp := do(t, http.MethodGet, srv.URL+"/bookmarks/1", ""); resp.StatusCode != http.StatusOK {
		t.Errorf("existing: got status %d, want 200", resp.StatusCode)
	}
	if resp := do(t, http.MethodGet, srv.URL+"/bookmarks/2", ""); resp.StatusCode != http.StatusNotFound {
		t.Errorf("missing: got status %d, want 404", resp.StatusCode)
	}
	if resp := do(t, http.MethodGet, srv.URL+"/bookmarks/abc", ""); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("bad id: got status %d, want 400", resp.StatusCode)
	}
}

func TestDelete(t *testing.T) {
	srv, s := newServer(t)
	s.Create(store.Bookmark{URL: "https://go.dev", Title: "Go"})

	if resp := do(t, http.MethodDelete, srv.URL+"/bookmarks/1", ""); resp.StatusCode != http.StatusNoContent {
		t.Errorf("got status %d, want 204", resp.StatusCode)
	}
	if resp := do(t, http.MethodDelete, srv.URL+"/bookmarks/1", ""); resp.StatusCode != http.StatusNotFound {
		t.Errorf("second delete: got status %d, want 404", resp.StatusCode)
	}
}
