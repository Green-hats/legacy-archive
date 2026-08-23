package server

import (
	"archive/zip"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"ani-rss/internal/log"
	"ani-rss/internal/service"
)

// handleLogs processes POST /api/logs.
func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	ok(w, service.GetLogs())
}

// handleClearLogs processes POST /api/clearLogs.
func (s *Server) handleClearLogs(w http.ResponseWriter, r *http.Request) {
	log.Clear()
	ok(w, nil)
}

// handleDownloadLogs serves GET /api/downloadLogs (zip of logs/*.log).
func (s *Server) handleDownloadLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `inline; filename="logs.zip"`)
	zw := zip.NewWriter(w)
	defer zw.Close()
	entries, err := os.ReadDir(log.LogsDir())
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".log") {
			continue
		}
		f, err := os.Open(filepath.Join(log.LogsDir(), e.Name()))
		if err != nil {
			continue
		}
		zf, err := zw.Create(e.Name())
		if err == nil {
			_, _ = io.Copy(zf, f)
		}
		f.Close()
	}
}