package download

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"ani-rss/internal/config"
	"ani-rss/internal/model"
	"ani-rss/internal/util"
)

func utilLogWarn(format string, args ...interface{}) { fmt.Printf("[WARN] "+format+"\n", args...) }
func utilLogErr(format string, args ...interface{})  { fmt.Printf("[ERROR] "+format+"\n", args...) }
func utilLogInfo(format string, args ...interface{}) { fmt.Printf("[INFO] "+format+"\n", args...) }

// GetMagnet reads the magnet/ed2k link from a saved torrent .txt file.
func GetMagnet(torrentPath string) string {
	b, err := os.ReadFile(torrentPath)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// dirOf returns the parent directory of a cloud/local path.
func dirOf(path string) string {
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[:i]
	}
	return ""
}

// baseOf returns the final name segment of a path.
func baseOf(path string) string {
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[i+1:]
	}
	return path
}

// Client is the normalized download-client interface (mirrors BaseDownload).
type Client interface {
	Login(test bool, cfg *model.Config) bool
	GetTorrentsInfos() []*model.TorrentsInfo
	Download(ani *model.Ani, item *model.Item, savePath string, torrentPath string) bool
	Delete(t *model.TorrentsInfo, deleteFiles bool) bool
	Rename(t *model.TorrentsInfo) bool
	AddTags(t *model.TorrentsInfo, tags string) bool
	UpdateTrackers(trackers []string)
	SetSavePath(t *model.TorrentsInfo, path string)
}

// GetFileReName computes the renamed file path for a file inside a torrent
// (video -> reName.ext, subtitle -> reName.lang.ext).
func GetFileReName(name, reName string) string {
	ext := strings.ToLower(filepath.Ext(name))
	if ext == "" {
		return name
	}
	ext = strings.TrimPrefix(ext, ".")
	newPath := reName
	if util.IsVideo(name) {
		newPath = newPath + "." + ext
	} else if util.IsSubtitle(name) {
		main := filepath.Base(name)
		main = strings.TrimSuffix(main, "."+ext)
		lang := filepath.Ext(main)
		if lang != "" {
			newPath = newPath + "." + strings.TrimPrefix(lang, ".")
		}
		newPath = newPath + "." + ext
	} else {
		return name
	}
	if name == newPath {
		return name
	}
	return newPath
}

// NewTags builds the tag list for a new download (ani-rss, subgroup, standby...).
func NewTags(ani *model.Ani, item *model.Item) []string {
	subgroup := item.Subgroup
	if strings.TrimSpace(subgroup) == "" {
		subgroup = "未知字幕组"
	}
	cfg := config.Get()
	tags := []string{model.TagAniRss, subgroup}
	if !item.Master {
		tags = append(tags, model.TagStandbyRss)
	}
	if ani.CustomTagsEnable {
		if len(ani.CustomTags) > 0 {
			tags = append(tags, ani.CustomTags...)
		}
		return tags
	}
	if len(cfg.CustomTags) > 0 {
		tags = append(tags, cfg.CustomTags...)
	}
	return tags
}

// NewTagsCsv joins tags with commas.
func NewTagsCsv(ani *model.Ani, item *model.Item) string {
	return strings.Join(NewTags(ani, item), ",")
}

// baseHost trims the trailing slash so host+path never becomes a double slash.
func baseHost(host string) string {
	return strings.TrimSuffix(host, "/")
}

// FindAniByDownloadPath is set by the service package to resolve the owning
// subscription of a torrent (avoids an import cycle).
var FindAniByDownloadPath func(t *model.TorrentsInfo) *model.Ani

// CloudClient is an optional capability of cloud-storage downloaders
// (PikPak / OpenList): files live remotely, so existence checks and playback
// URLs must go through the cloud API instead of the local filesystem.
type CloudClient interface {
	// FileExists reports whether a remote file exists at the cloud path
	// (absolute cloud path built from the download path template + reName).
	FileExists(cloudPath string) bool
	// FileURL returns a playable/download URL for the remote file.
	FileURL(cloudPath string) string
	// ListDir returns the files in a cloud directory (for the play list).
	ListDir(cloudPath string) []CloudFile
}

// CloudFile is one entry returned by ListDir.
type CloudFile struct {
	Name     string
	Size     int64
	IsDir    bool
	ID       string // provider-specific id (115 cid/fid)
	PickCode string // provider-specific download code (115)
}

// onceWarned tracks which config-incomplete warnings have already been logged.
var (
	onceWarnedMu sync.Mutex
	onceWarned   = map[string]bool{}
)

// warnOnce logs a warning only once per key until ResetWarnings is called
// (on config change). Prevents the "xx 未配置完成" flood from task loops.
func warnOnce(key, format string, args ...interface{}) {
	onceWarnedMu.Lock()
	if onceWarned[key] {
		onceWarnedMu.Unlock()
		return
	}
	onceWarned[key] = true
	onceWarnedMu.Unlock()
	utilLogWarn(format, args...)
}

// ResetWarnings clears the once-logged warning markers (called on reload).
func ResetWarnings() {
	onceWarnedMu.Lock()
	for k := range onceWarned {
		delete(onceWarned, k)
	}
	onceWarnedMu.Unlock()
}