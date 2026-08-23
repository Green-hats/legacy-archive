package server

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"ani-rss/internal/config"
	"ani-rss/internal/download"
	"ani-rss/internal/matroska"
	"ani-rss/internal/model"
	"ani-rss/internal/service"
	"ani-rss/internal/util"
)

// handleFile serves GET /api/file with Range support. For cloud downloaders
// (PikPak) the path resolves to the cloud and is served via a redirect to the
// cloud media URL.
func (s *Server) handleFile(w http.ResponseWriter, r *http.Request) {
	b64 := r.URL.Query().Get("filename")
	if b64 == "" {
		http.Error(w, "filename is required", http.StatusBadRequest)
		return
	}
	path, ok := resolveFilePath(b64)
	if ok {
		if fi, err := os.Stat(path); err == nil && !fi.IsDir() {
			ext := strings.ToLower(filepath.Ext(path))
			if util.IsVideo(path) || util.IsSubtitle(path) || ext == ".jpg" || ext == ".png" || ext == ".webp" || ext == ".gif" {
				w.Header().Set("Content-Type", contentTypeFor(path))
				w.Header().Set("Content-Disposition", "inline")
				if util.IsVideo(path) || fi.Size() > 3*1024*1024 {
					w.Header().Set("Cache-Control", "no-store")
				} else {
					w.Header().Set("Cache-Control", "public, max-age=2592000")
				}
				http.ServeFile(w, r, path)
				return
			}
		}
	}
	// cloud downloader fallback: proxy the cloud media stream through this
	// server (supplies the provider's UA and forwards Range), so both the web
	// player and external players (MPV) can play without 115 auth/UA issues.
	if _, ok := download.Type().(download.CloudClient); ok {
		raw, ok2 := decodeBase64Path(b64)
		if ok2 && raw != "" {
			download.ProxyCloudFile(w, r, raw)
			return
		}
	}
	http.NotFound(w, r)
}

// handleCalendarIcs serves GET /api/calendar.ics.
func (s *Server) handleCalendarIcs(w http.ResponseWriter, r *http.Request) {
	var sb strings.Builder
	sb.WriteString("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//ani-rss//ani-rss//EN\r\n")
	for _, ani := range config.AniList() {
		if ani == nil || ani.ReleaseDate.Time().IsZero() {
			continue
		}
		d := ani.ReleaseDate.Time()
		sb.WriteString("BEGIN:VEVENT\r\n")
		sb.WriteString("SUMMARY:" + ani.Title + "\r\n")
		sb.WriteString("DTSTART;VALUE=DATE:" + d.Format("20060102") + "\r\n")
		sb.WriteString("END:VEVENT\r\n")
	}
	sb.WriteString("END:VCALENDAR\r\n")
	w.Header().Set("Content-Type", "text/calendar; charset=UTF-8")
	w.Header().Set("Content-Disposition", `attachment; filename=ani-rss-calendar.ics`)
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write([]byte(sb.String()))
}

// handleProxyImage serves GET /api/proxyImage.
func (s *Server) handleProxyImage(w http.ResponseWriter, r *http.Request) {
	b64 := r.URL.Query().Get("imgUrl")
	if b64 == "" {
		http.Error(w, "imgUrl is required", http.StatusBadRequest)
		return
	}
	safe := strings.ReplaceAll(b64, " ", "+")
	decoded, err := base64.StdEncoding.DecodeString(safe)
	if err != nil {
		decoded, err = base64.URLEncoding.DecodeString(safe)
		if err != nil {
			http.Error(w, "bad base64", http.StatusBadRequest)
			return
		}
	}
	imgURL := string(decoded)
	hash := util.MD5Hex(imgURL)
	ext := extFromURL(imgURL)
	if ext == "" {
		ext = ".jpg"
	}
	cachePath := config.ConfigDirFile("img/" + hash + ext)
	if b, err := os.ReadFile(cachePath); err == nil {
		w.Header().Set("Content-Type", contentTypeFor(cachePath))
		w.Header().Set("Cache-Control", "public, max-age=2592000")
		w.Write(b)
		return
	}
	b, err := util.GetBytes(imgURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err == nil {
		_ = os.WriteFile(cachePath, b, 0o644)
	}
	w.Header().Set("Content-Type", contentTypeFor(cachePath))
	w.Header().Set("Cache-Control", "public, max-age=2592000")
	w.Write(b)
}

// handleTorrentsInfos processes POST /api/torrentsInfos.
func (s *Server) handleTorrentsInfos(w http.ResponseWriter, r *http.Request) {
	ok(w, download.GetTorrentsInfos())
}

// handleDeleteTorrent processes POST /api/deleteTorrent.
func (s *Server) handleDeleteTorrent(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	hash := r.URL.Query().Get("hash")
	if id == "" || hash == "" {
		fail(w, "id 和 hash 不能为空")
		return
	}
	exists := false
	for _, ani := range config.AniList() {
		if ani != nil && ani.ID == id {
			exists = true
			break
		}
	}
	if !exists {
		fail(w, "此订阅不存在")
		return
	}
	for _, h := range strings.Split(hash, ",") {
		h = strings.TrimSpace(h)
		if t := download.FindByHash(h); t != nil {
			service.DeleteTorrent(t, true, false)
		}
	}
	okMsg(w, "删除完成")
}

// handlePlayList processes POST /api/playList.
func (s *Server) handlePlayList(w http.ResponseWriter, r *http.Request) {
	var body model.Ani
	if !readJSONOrFail(w, r, &body) {
		return
	}
	found := false
	for _, ani := range config.AniList() {
		if ani != nil && ani.URL == body.URL {
			found = true
			break
		}
	}
	if !found {
		fail(w, "此订阅不存在")
		return
	}
	ok(w, buildPlayList(body))
}

func buildPlayList(ani model.Ani) []*model.PlayItem {
	dir := service.GetDownloadPath(&ani)
	var out []*model.PlayItem
	// cloud downloaders (PikPak / 115): list files from the cloud directory,
	// recursing into subdirectories (115 offline wraps each file in a folder).
	if cloud, ok := download.Type().(download.CloudClient); ok {
		var walk func(path string, depth int)
		walk = func(path string, depth int) {
			if depth > 5 {
				return
			}
			for _, f := range cloud.ListDir(path) {
				if f.IsDir {
					walk(path+"/"+f.Name, depth+1)
					continue
				}
				if !util.IsVideo(f.Name) {
					continue
				}
				cloudPath := path + "/" + f.Name
				item := &model.PlayItem{
					Title:      f.Name,
					Filename:   cloudPath,
					Name:       f.Name,
					LastModify: 0,
					Episode:    extractPlayEpisode(f.Name),
					FormatSize: util.FormatSize(f.Size),
					ExtName:    strings.TrimPrefix(filepath.Ext(f.Name), "."),
					Subtitles:  []model.PlaySubtitle{},
				}
				out = append(out, item)
			}
		}
		walk(dir, 0)
		return out
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return out
	}
	for _, e := range entries {
		if e.IsDir() || !util.IsVideo(e.Name()) {
			continue
		}
		full := filepath.Join(dir, e.Name())
		fi, err := e.Info()
		if err != nil {
			continue
		}
		ep := extractPlayEpisode(e.Name())
		item := &model.PlayItem{
			Title:      e.Name(),
			Filename:   full,
			Name:       e.Name(),
			LastModify: fi.ModTime().UnixMilli(),
			Episode:    ep,
			FormatSize: util.FormatSize(fi.Size()),
			ExtName:    strings.TrimPrefix(filepath.Ext(e.Name()), "."),
		}
		item.Subtitles = findSubtitles(dir, e.Name())
		out = append(out, item)
	}
	return out
}

func extractPlayEpisode(name string) float64 {
	m := service.SeasonEpisodeRe.FindStringSubmatch(name)
	if len(m) > 2 {
		f, _ := strconv.ParseFloat(m[2], 64)
		return f
	}
	return 0
}

func findSubtitles(dir, videoName string) []model.PlaySubtitle {
	out := []model.PlaySubtitle{}
	base := strings.TrimSuffix(videoName, filepath.Ext(videoName))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return out
	}
	for _, e := range entries {
		if e.IsDir() || !util.IsSubtitle(e.Name()) {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, base) {
			full := filepath.Join(dir, name)
			if b, err := os.ReadFile(full); err == nil {
				out = append(out, model.PlaySubtitle{
					HTML:    fmt.Sprintf("<track src=\"%s\">", full),
					Name:    name,
					URL:     full,
					Content: string(b),
					Type:    "vtt",
				})
			}
		}
	}
	return out
}

// handleGetSubtitles processes POST /api/getSubtitles (mkv embedded subs).
func (s *Server) handleGetSubtitles(w http.ResponseWriter, r *http.Request) {
	b64 := r.URL.Query().Get("filename")
	if b64 == "" {
		fail(w, "filename 不能为空")
		return
	}
	path, valid := resolveFilePath(b64)
	if !valid || strings.ToLower(filepath.Ext(path)) != ".mkv" {
		fail(w, "仅支持 mkv 文件")
		return
	}
	// cloud downloaders (115 / PikPak): the file lives remotely, so embedded
	// subs can't be read locally — return an empty list instead of failing.
	if fi, err := os.Stat(path); err != nil || fi.IsDir() {
		ok(w, []model.PlaySubtitle{})
		return
	}
	subs, err := matroska.ExtractSubtitles(path)
	if err != nil {
		fail(w, err.Error())
		return
	}
	var out []model.PlaySubtitle
	for _, sub := range subs {
		name := sub.Name
		if name == "" {
			name = "embedded"
		}
		if sub.Language != "" {
			name += " [" + sub.Language + "]"
		}
		out = append(out, model.PlaySubtitle{
			HTML:    strings.ToUpper(name),
			Name:    name,
			URL:     "",
			Content: sub.Content,
			Type:    "vtt",
		})
	}
	ok(w, out)
}

// handleUpload processes POST /api/upload.
func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		fail(w, err.Error())
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		fail(w, "未获取到文件")
		return
	}
	defer file.Close()
	typ := r.URL.Query().Get("type")
	if typ == "getBase64" {
		b, _ := io.ReadAll(file)
		ok(w, base64.StdEncoding.EncodeToString(b))
		return
	}
	b, _ := io.ReadAll(file)
	hash := util.MD5Hex(string(b))
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext == "" {
		ext = ""
	}
	rel := filepath.ToSlash(filepath.Join(hash[:1], hash+ext))
	full := config.ConfigDirFile("files/" + rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		fail(w, err.Error())
		return
	}
	if err := os.WriteFile(full, b, 0o644); err != nil {
		fail(w, err.Error())
		return
	}
	writeResult(w, &model.Result{Code: 200, Message: "上传完成", Data: rel, T: model.Now().UnixMilli()})
}

var _ = time.Now