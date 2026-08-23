package service

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ani-rss/internal/config"
	"ani-rss/internal/download"
	"ani-rss/internal/log"
	"ani-rss/internal/model"
	"ani-rss/internal/util"
)

// SetConfigRaw merges the raw JSON body into the current config.
func SetConfigRaw(raw []byte) error {
	cur := config.Get()
	oldRssSleep := cur.RssSleepMinutes
	oldRenameSleep := cur.RenameSleepSeconds
	oldTool := cur.DownloadToolType
	if err := config.MergeConfigInto(cur, raw); err != nil {
		return err
	}
	if err := config.Sync(); err != nil {
		return err
	}
	// restart task loops when timing changed
	if oldRssSleep != cur.RssSleepMinutes || oldRenameSleep != cur.RenameSleepSeconds {
		RestartTasks()
	}
	// rebuild download client when tool changed
	if oldTool != cur.DownloadToolType {
		download.Reload()
	}
	return nil
}

// DownloadLoginTest attempts to log into the configured download client.
func DownloadLoginTest() error {
	if !download.Login(true) {
		return errors.New("登录失败")
	}
	return nil
}

// TestProxy tests the configured proxy against a URL.
func TestProxy(rawURL string, cfg *model.Config) (*model.ProxyTest, error) {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", util.UserAgent())
	client := util.ClientFor(20)
	if cfg.Proxy {
		transport := client.Transport.(*http.Transport)
		transport.Proxy = http.ProxyURL(&url.URL{
			Scheme: "http",
			Host:   cfg.ProxyHost,
		})
		if cfg.ProxyUsername != "" {
			transport.Proxy = http.ProxyURL(&url.URL{
				Scheme: "http", Host: cfg.ProxyHost,
				User: url.UserPassword(cfg.ProxyUsername, cfg.ProxyPassword),
			})
		}
	}
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return &model.ProxyTest{
		Status: resp.StatusCode,
		Time:   time.Since(start).Milliseconds(),
	}, nil
}

// UpdateTrackers pushes the tracker list to the download client.
func UpdateTrackers(cfg *model.Config) {
	urls := cfg.TrackersUpdateUrls
	var trackers []string
	for _, line := range strings.Split(urls, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if trackers, err := fetchTrackers(line); err == nil {
			for _, t := range trackers {
				if !containsStr(trackers, t) {
					trackers = append(trackers, t)
				}
			}
		}
	}
	if len(trackers) > 0 {
		download.Type().UpdateTrackers(trackers)
	}
}

func fetchTrackers(rawURL string) ([]string, error) {
	b, err := util.GetBytes(rawURL)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "http") {
			out = append(out, line)
		}
	}
	return out, nil
}

// ClearCache clears the in-memory TTL cache and the img cache dir.
func ClearCache() (string, error) {
	size := cacheSize()
	configDir := config.Dir()
	imgDir := filepath.Join(configDir, "img")
	if err := os.RemoveAll(imgDir); err != nil {
		return "", err
	}
	ClearInMemoryCache()
	return fmt.Sprintf("清理完成, 共清理 %d", size), nil
}

// writeBackupZip writes files/, torrents/, ani.v2.json, config.v2.json to a zip.
func writeBackupZip(outPath string) error {
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	defer zw.Close()
	configDir := config.Dir()
	addZipDir := func(dir string) error {
		return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			if strings.HasPrefix(filepath.Base(path), ".") {
				return nil
			}
			rel, err := filepath.Rel(configDir, path)
			if err != nil {
				return nil
			}
			rel = filepath.ToSlash(rel)
			w, err := zw.Create(rel)
			if err != nil {
				return err
			}
			src, err := os.Open(path)
			if err != nil {
				return nil
			}
			_, _ = io.Copy(w, src)
			src.Close()
			return nil
		})
	}
	if err := addZipDir(filepath.Join(configDir, "files")); err != nil {
		return err
	}
	if err := addZipDir(filepath.Join(configDir, "torrents")); err != nil {
		return err
	}
	for _, name := range []string{config.AniFile, config.ConfigFile} {
		p := filepath.Join(configDir, name)
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		src, err := os.Open(p)
		if err != nil {
			continue
		}
		_, _ = io.Copy(w, src)
		src.Close()
	}
	return nil
}

// ExportConfig streams a backup zip to w.
func ExportConfig(w io.Writer) error {
	tmp := filepath.Join(os.TempDir(), "ani-rss-backup-"+time.Now().Format("20060102150405")+".zip")
	if err := writeBackupZip(tmp); err != nil {
		return err
	}
	defer os.Remove(tmp)
	f, err := os.Open(tmp)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(w, f)
	return err
}

// ImportConfig restores config.v2.json and ani.v2.json from a backup zip.
func ImportConfig(zipPath string) error {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer zr.Close()
	configDir := config.Dir()
	for _, f := range zr.File {
		name := filepath.Base(f.Name)
		if name != config.ConfigFile && name != config.AniFile {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		out, err := os.Create(filepath.Join(configDir, name))
		if err != nil {
			rc.Close()
			return err
		}
		_, _ = io.Copy(out, rc)
		rc.Close()
		out.Close()
	}
	return nil
}

// cacheSize is replaced by the server package.
var cacheSize = func() int { return 0 }

// SetCacheSizeFunc wires the cache-size reporter.
func SetCacheSizeFunc(fn func() int) { cacheSize = fn }

// ClearInMemoryCache is wired to the TTL cache.
var ClearInMemoryCache = func() {}

// RestartTasks is wired to the task manager.
var RestartTasks = func() {}

// GetLogs returns the in-memory log ring buffer.
func GetLogs() []model.Log { return log.List() }