// Package web embeds the compiled front end (web/dist) into the Go binary and
// serves it.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// distFS returns the contents of web/dist, as produced by `npm run build`.
func distFS() fs.FS {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err)
	}
	return sub
}
