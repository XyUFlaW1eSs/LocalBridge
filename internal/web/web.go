package web

import (
	"embed"
	"io"
	"mime"
	"net/http"
	"path"
	"strings"
)

//go:embed static/*
var staticFiles embed.FS

func Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /", redirectRoot)
	mux.HandleFunc("GET /app", redirectApp)
	mux.HandleFunc("GET /app/{asset...}", serveAsset)
}

func redirectRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		http.Redirect(w, r, "/app/", http.StatusTemporaryRedirect)
		return
	}
	http.NotFound(w, r)
}

func redirectApp(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/app/", http.StatusTemporaryRedirect)
}

func serveIndex(w http.ResponseWriter, r *http.Request) { serve(w, r, "static/index.html") }

func serveAsset(w http.ResponseWriter, r *http.Request) {
	asset := strings.TrimPrefix(r.PathValue("asset"), "/")
	asset = path.Clean(asset)
	if asset == "." || asset == "" {
		serveIndex(w, r)
		return
	}
	if asset == "." || asset == ".." || strings.HasPrefix(asset, "../") || strings.Contains(asset, "\\") {
		http.NotFound(w, r)
		return
	}
	serve(w, r, "static/"+asset)
}

func serve(w http.ResponseWriter, r *http.Request, name string) {
	file, err := staticFiles.Open(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}
	if contentType := mime.TypeByExtension(path.Ext(name)); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	w.Header().Set("Cache-Control", "no-store")
	reader, ok := file.(io.ReadSeeker)
	if !ok {
		http.NotFound(w, r)
		return
	}
	http.ServeContent(w, r, info.Name(), info.ModTime(), reader)
}
