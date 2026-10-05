package embed_test

import (
	"bytes"
	"image/png"
	"strings"
	"testing"

	"github.com/gz-zenn/go-annotations/embed"
)

func TestDefaultConfigIsEmbeddedAsString(t *testing.T) {
	got := embed.DefaultConfig()

	if got == "" {
		t.Fatal("DefaultConfig() returned an empty string")
	}
	for _, want := range []string{"server:", "addr:", "logging:"} {
		if !strings.Contains(got, want) {
			t.Errorf("embedded config is missing %q\ngot:\n%s", want, got)
		}
	}
}

func TestLogoIsEmbeddedAsBytesAndDecodes(t *testing.T) {
	logo := embed.Logo()
	if len(logo) == 0 {
		t.Fatal("Logo() returned no bytes")
	}

	// A real PNG signature, then a real decode: the bytes came from the
	// binary, not from the working directory.
	if !bytes.HasPrefix(logo, []byte("\x89PNG\r\n\x1a\n")) {
		t.Fatalf("logo does not start with the PNG signature: % x", logo[:8])
	}

	img, err := png.Decode(bytes.NewReader(logo))
	if err != nil {
		t.Fatalf("png.Decode: %v", err)
	}
	if got := img.Bounds().Dx(); got != 16 {
		t.Errorf("logo width = %d, want 16", got)
	}
}

func TestWebAssetsCombinesSeveralPatterns(t *testing.T) {
	assets := embed.WebAssets()

	want := []string{
		"templates/page.html",
		"static/css/app.css",
		"static/js/app.js",
	}
	for _, name := range want {
		if _, err := assets.Open(name); err != nil {
			t.Errorf("Open(%s): %v", name, err)
		}
	}

	// A pattern only matches what it names: the second level of static is
	// outside the pattern, and the config file is not in the list.
	for _, name := range []string{"static/sub/nested.txt", "static/index.html", "config/default.yaml"} {
		if _, err := assets.Open(name); err == nil {
			t.Errorf("Open(%s) succeeded, want an error: the pattern does not match it", name)
		}
	}
}
