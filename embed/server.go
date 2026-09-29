// Package embed demonstrates //go:embed, the directive that compiles files
// into the binary at build time. Nothing is read from disk at runtime and
// nothing can go missing in production, because the bytes are part of the
// executable.
//
// Every target must be a package-level variable of type string, []byte or
// embed.FS, and the embed package must be imported (even blank, when only
// the side effect is needed).
package embed

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static
var staticFiles embed.FS

// StaticHandler serves the embedded static directory from the site root.
//
// fs.Sub is what makes the paths work: an embed.FS keeps the directory it was
// given, so static/app.css lives at "static/app.css" inside the FS. Serving
// the FS directly at "/" would answer GET /app.css with 404, and the file
// would only be reachable at /static/app.css.
func StaticHandler() (http.Handler, error) {
	sub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return nil, err
	}
	return http.FileServer(http.FS(sub)), nil
}

// RawHandler is the version that does not strip the prefix. It is here
// because it is the mistake everybody makes first: it starts, it serves
// something, and every asset is 404 unless the URL repeats "static".
func RawHandler() http.Handler {
	return http.FileServer(http.FS(staticFiles))
}

// StaticFS exposes the whole embedded tree, prefix included.
func StaticFS() embed.FS {
	return staticFiles
}
