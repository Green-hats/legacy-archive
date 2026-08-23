package service

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"ani-rss/internal/config"
	"ani-rss/internal/model"
	"ani-rss/internal/util"
)

var (
	regMagnet = regexp.MustCompile(`^magnet:\?xt=urn:btih:(\w+)`)
	regEd2k   = regexp.MustCompile(`^ed2k://\|file\|([^|]+)\|(\d+)\|([A-Fa-f0-9]{32})\|/$`)
)

// TorrentDir returns the torrent cache folder for an ani.
func TorrentDir(ani *model.Ani) string {
	cfgDir := config.Dir()
	title := ani.Title
	season := ani.Season
	letter := util.GetPinyinInitials(title)

	base := filepath.Join(cfgDir, "torrents", title, "Season "+itoa(season))
	if _, err := os.Stat(base); err != nil {
		base = filepath.Join(cfgDir, "torrents", letter, title, "Season "+itoa(season))
	}
	if ani.Ova {
		base = filepath.Join(cfgDir, "torrents", title)
		if _, err := os.Stat(base); err != nil {
			base = filepath.Join(cfgDir, "torrents", letter, title)
		}
	}
	return base
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}

// TorrentFile returns the cached torrent file path for an item.
func TorrentFile(ani *model.Ani, item *model.Item) string {
	dir := TorrentDir(ani)
	if regMagnet.MatchString(item.Torrent) || regEd2k.MatchString(item.Torrent) {
		return filepath.Join(dir, item.InfoHash+".txt")
	}
	return filepath.Join(dir, item.InfoHash+".torrent")
}

// SaveTorrent downloads/saves the torrent file to the cache dir.
func SaveTorrent(ani *model.Ani, item *model.Item) string {
	torrent := item.Torrent
	path := TorrentFile(ani, item)
	if fi, err := os.Stat(path); err == nil && fi.Size() >= 0 {
		return path
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return path
	}
	if regMagnet.MatchString(torrent) || regEd2k.MatchString(torrent) {
		_ = os.WriteFile(path, []byte(torrent), 0o644)
		return path
	}
	// download .torrent
	resp, err := util.Get(torrent)
	if err != nil {
		_ = os.Remove(path)
		return path
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		_ = os.WriteFile(path, nil, 0o644)
		return path
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_ = os.Remove(path)
		return path
	}
	b := make([]byte, 0)
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return path
	}
	if _, err := f.ReadFrom(resp.Body); err != nil {
		f.Close()
		os.Remove(tmp)
		return path
	}
	f.Close()
	os.Rename(tmp, path)
	_ = b
	return path
}

// GetMagnet builds a magnet link from a cached torrent file.
func GetMagnet(path string) string {
	hash := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	fi, err := os.Stat(path)
	if err != nil || fi.Size() < 1 {
		return "magnet:?xt=urn:btih:" + hash
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".txt" {
		b, _ := os.ReadFile(path)
		return strings.TrimSpace(string(b))
	}
	// .torrent: fall back to filename-derived hash
	return "magnet:?xt=urn:btih:" + hash
}

// AllowDelete decides whether a finished torrent can be deleted.
func AllowDelete(t *model.TorrentsInfo) bool {
	cfg := config.Get()
	if cfg.AwaitStalledUP {
		return t.State == model.StateStoppedUP
	}
	return t.Finished()
}