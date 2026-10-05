package embed_test

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/gz-zenn/go-annotations/embed"
)

func TestStaticHandlerServesFromTheSiteRoot(t *testing.T) {
	handler, err := embed.StaticHandler()
	if err != nil {
		t.Fatalf("StaticHandler: %v", err)
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	for _, path := range []string{"/", "/index.html", "/css/app.css", "/js/app.js"} {
		t.Run(path, func(t *testing.T) {
			resp, err := http.Get(server.URL + path)
			if err != nil {
				t.Fatalf("GET %s: %v", path, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("GET %s = %d, want 200", path, resp.StatusCode)
			}
		})
	}
}

func TestStaticHandlerServesTheRightContent(t *testing.T) {
	handler, err := embed.StaticHandler()
	if err != nil {
		t.Fatalf("StaticHandler: %v", err)
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Get(server.URL + "/css/app.css")
	if err != nil {
		t.Fatalf("GET /css/app.css: %v", err)
	}
	defer resp.Body.Close()

	body := make([]byte, 256)
	n, _ := resp.Body.Read(body)
	if !strings.Contains(string(body[:n]), "font-family") {
		t.Errorf("body = %q, want it to contain the stylesheet", body[:n])
	}
}

func TestRawHandlerKeepsThePrefix(t *testing.T) {
	// This is the mistake the fs.Sub call exists to avoid. Nothing warns at
	// build time and nothing panics at start: the server happily answers, and
	// the assets are only reachable under a URL that repeats "static".
	server := httptest.NewServer(embed.RawHandler())
	defer server.Close()

	// Redirects are followed here, so the paths are checked as a browser
	// would see them.
	client := server.Client()

	tests := []struct {
		path string
		want int
	}{
		{"/static/index.html", http.StatusOK},
		{"/static/css/app.css", http.StatusOK},
		{"/css/app.css", http.StatusNotFound},
		{"/index.html", http.StatusOK},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			resp, err := client.Get(server.URL + tc.path)
			if err != nil {
				t.Fatalf("GET %s: %v", tc.path, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.want {
				t.Errorf("GET %s = %d, want %d", tc.path, resp.StatusCode, tc.want)
			}
		})
	}
}

func TestRawHandlerRedirectsIndexRequests(t *testing.T) {
	// http.FileServer maps any .../index.html back to its directory, which is
	// why the un-prefixed handler answers /index.html with a listing of the
	// package directory rather than with the page.
	server := httptest.NewServer(embed.RawHandler())
	defer server.Close()

	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	for _, path := range []string{"/index.html", "/static/index.html"} {
		t.Run(path, func(t *testing.T) {
			resp, err := client.Get(server.URL + path)
			if err != nil {
				t.Fatalf("GET %s: %v", path, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusMovedPermanently {
				t.Errorf("GET %s = %d, want 301", path, resp.StatusCode)
			}
		})
	}
}

func TestRawHandlerServesADirectoryListingAtTheRoot(t *testing.T) {
	// The site root of the un-prefixed handler is the package directory, so
	// what a browser gets at / is a listing, not the page.
	server := httptest.NewServer(embed.RawHandler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()

	body := make([]byte, 4096)
	n, _ := resp.Body.Read(body)
	listing := string(body[:n])

	if strings.Contains(listing, "served from an embed.FS") {
		t.Error("root served index.html, want a directory listing")
	}
	if !strings.Contains(listing, "static/") {
		t.Errorf("root body = %q, want a directory listing containing static/", listing)
	}
}

func TestEmbeddedFileKeepsItsPathInsideTheFS(t *testing.T) {
	// The reason fs.Sub is needed: embed.FS stores package-relative paths.
	if _, err := embed.StaticFS().Open("static/index.html"); err != nil {
		t.Errorf("Open(static/index.html): %v", err)
	}
	if _, err := embed.StaticFS().Open("index.html"); err == nil {
		t.Error("Open(index.html) succeeded, want an error: the prefix is part of the name")
	}
}

func TestFsSubIsWhatMakesTheRootWork(t *testing.T) {
	sub, err := fs.Sub(embed.StaticFS(), "static")
	if err != nil {
		t.Fatalf("fs.Sub: %v", err)
	}

	if _, err := sub.Open("index.html"); err != nil {
		t.Errorf("sub.Open(index.html): %v", err)
	}
	if _, err := sub.Open("static/index.html"); err == nil {
		t.Error("sub.Open(static/index.html) succeeded, want an error")
	}
}

func TestDotfileIsNotServedByDefault(t *testing.T) {
	// static/sub/.env is skipped by the "static" pattern, so the file cannot
	// be fetched even though it sits in the served directory on disk. This is
	// the reason to prefer the bare directory over static/* or all:static.
	handler, err := embed.StaticHandler()
	if err != nil {
		t.Fatalf("StaticHandler: %v", err)
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	for _, path := range []string{"/sub/.env", "/.nojekyll"} {
		t.Run(path, func(t *testing.T) {
			resp, err := http.Get(server.URL + path)
			if err != nil {
				t.Fatalf("GET %s: %v", path, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusNotFound {
				t.Errorf("GET %s = %d, want 404", path, resp.StatusCode)
			}
		})
	}

	// The rest of the directory is served normally, so the 404s above are
	// about the dotfile rule and not about the path being wrong.
	resp, err := http.Get(server.URL + "/sub/nested.txt")
	if err != nil {
		t.Fatalf("GET /sub/nested.txt: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /sub/nested.txt = %d, want 200", resp.StatusCode)
	}
}

func TestFileListIsSorted(t *testing.T) {
	// A guard on the shared test data: several assertions below depend on
	// the exact file set, so a stray file in static/ would make them lie.
	var paths []string
	err := fs.WalkDir(embed.StaticFS(), "static", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir: %v", err)
	}
	sort.Strings(paths)

	want := []string{
		"static/css/app.css",
		"static/index.html",
		"static/js/app.js",
		"static/sub/nested.txt",
	}

	if strings.Join(paths, ",") != strings.Join(want, ",") {
		t.Errorf("embedded files = %v, want %v", paths, want)
	}
}
