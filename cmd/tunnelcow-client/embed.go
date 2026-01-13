package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
)

var webDist embed.FS

func getWebFileSystem() http.FileSystem {

	fsys, err := fs.Sub(webDist, "web_dist")
	if err != nil {
		log.Fatal(err)
	}
	return http.FS(fsys)
}
