package server

import (
	"encoding/base64"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"ani-rss/internal/config"
	"ani-rss/internal/service"
	"ani-rss/internal/util"
)

// saveCover downloads an image url and stores it under files/ returning the
// relative path "<md5[0]>/<md5>.<ext>".
func saveCover(imageURL string) string {
	return service.SaveCover(imageURL)
}

func extFromURL(rawURL string) string {
	path := rawURL
	if i := strings.Index(path, "?"); i >= 0 {
		path = path[:i]
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".webp" || ext == ".gif" {
		if ext == ".jpeg" {
			ext = ".jpg"
		}
		return ext
	}
	return ""
}

// resolveFilePath resolves a base64-encoded filename path for /api/file.
// Mirrors Java FileController: absolute paths are used as-is; relative cover /
// upload paths resolve under <configDir>/files/.
func resolveFilePath(b64 string) (string, bool) {
	decoded, ok := decodeBase64Path(b64)
	if !ok {
		return "", false
	}
	p := decoded
	cfgDir := config.Dir()
	if filepath.IsAbs(p) {
		return p, true
	}
	if strings.HasPrefix(p, cfgDir) {
		return p, true
	}
	// cover / upload files live under <configDir>/files/<rel>
	filesCandidate := filepath.Join(cfgDir, "files", p)
	if _, err := os.Stat(filesCandidate); err == nil {
		return filesCandidate, true
	}
	return filepath.Join(cfgDir, p), true
}

// decodeBase64Path decodes a base64 filename query value.
func decodeBase64Path(b64 string) (string, bool) {
	s := strings.ReplaceAll(b64, " ", "+")
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		decoded, err = base64.URLEncoding.DecodeString(s)
		if err != nil {
			return "", false
		}
	}
	return string(decoded), true
}

func contentTypes() map[string]string {
	return map[string]string{
		".html": "text/html", ".css": "text/css", ".js": "application/javascript",
		".json": "application/json", ".png": "image/png", ".jpg": "image/jpeg",
		".jpeg": "image/jpeg", ".gif": "image/gif", ".webp": "image/webp",
		".svg": "image/svg+xml", ".ico": "image/x-icon", ".bmp": "image/bmp",
		".mp4": "video/mp4", ".mkv": "video/x-matroska", ".avi": "video/x-msvideo",
		".wmv": "video/x-ms-wmv", ".mov": "video/quicktime", ".ts": "video/mp2t",
		".flv": "video/x-flv", ".webm": "video/webm",
		".ass": "text/plain", ".ssa": "text/plain", ".srt": "text/plain", ".vtt": "text/vtt",
		".nfo": "text/plain", ".txt": "text/plain", ".ini": "text/plain", ".xml": "text/xml",
	}
}

func contentTypeFor(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ct := mime.TypeByExtension(ext); ct != "" {
		return ct
	}
	if ct, ok := contentTypes()[ext]; ok {
		return ct
	}
	if util.IsVideo(path) {
		return "video/*"
	}
	return "application/octet-stream"
}