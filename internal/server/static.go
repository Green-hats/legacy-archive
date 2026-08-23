package server

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"ani-rss/internal/config"
)

//go:embed all:webui
var webuiFS embed.FS

// staticFileServer serves the built webui. Embedded assets take precedence;
// files in <configDir>/webui override them (allows user customization).
func staticFileServer() http.HandlerFunc {
	embedded, err := fs.Sub(webuiFS, "webui")
	if err != nil {
		embedded = nil
	}
	return func(w http.ResponseWriter, r *http.Request) {
		cfgDir := config.Dir()
		rel := strings.TrimPrefix(r.URL.Path, "/")
		if rel == "" {
			rel = "index.html"
		}
		// SPA fallback: extension-less paths map to <path>.html
		if !strings.Contains(path.Base(rel), ".") {
			rel = rel + ".html"
		}

		// 1. user override dir
		userPath := filepath.Join(cfgDir, "webui", filepath.FromSlash(rel))
		if fi, err := os.Stat(userPath); err == nil && !fi.IsDir() {
			http.ServeFile(w, r, userPath)
			return
		}

		// 2. embedded assets
		if embedded != nil {
			if b, err := fs.ReadFile(embedded, rel); err == nil {
				w.Header().Set("Content-Type", contentTypeFor(rel))
				w.Header().Set("Cache-Control", cacheControlFor(rel))
				w.Write(b)
				return
			}
			// SPA fallback to index.html
			if b, err := fs.ReadFile(embedded, "index.html"); err == nil {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Write(b)
				return
			}
		}

		http.NotFound(w, r)
	}
}

func cacheControlFor(rel string) string {
	if strings.HasPrefix(rel, "assets/") {
		return "public, max-age=31536000, immutable"
	}
	return "no-cache"
}