package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"ani-rss/internal/bgm"
	"ani-rss/internal/config"
	"ani-rss/internal/download"
	"ani-rss/internal/log"
	"ani-rss/internal/model"
	"ani-rss/internal/rename"
	"ani-rss/internal/rss"
	"ani-rss/internal/util"
)

// ListAni returns the grouped subscription list sorted per config.
func ListAni() *model.ListAni {
	cfg := config.Get()
	list := config.AniList()
	sortBy := cfg.SortType

	sorted := append([]*model.Ani(nil), list...)
	for _, a := range sorted {
		if a == nil {
			continue
		}
		a.Pinyin = util.GetPinyin(a.Title)
		a.PinyinInitials = util.GetPinyinInitials(a.Title)
	}
	switch sortBy {
	case "PINYIN":
		sort.SliceStable(sorted, func(i, j int) bool {
			if sorted[i] == nil || sorted[j] == nil {
				return false
			}
			return sorted[i].Pinyin < sorted[j].Pinyin
		})
	case "DOWNLOAD_TIME":
		sort.SliceStable(sorted, func(i, j int) bool {
			if sorted[i] == nil || sorted[j] == nil {
				return false
			}
			return sorted[i].LastDownloadTime > sorted[j].LastDownloadTime
		})
	default: // SCORE
		sort.SliceStable(sorted, func(i, j int) bool {
			if sorted[i] == nil || sorted[j] == nil {
				return false
			}
			return sorted[i].Score > sorted[j].Score
		})
	}

	weeks := []string{"星期日", "星期一", "星期二", "星期三", "星期四", "星期五", "星期六"}
	weekItems := map[string][]*model.Ani{}
	for _, w := range weeks {
		weekItems[w] = []*model.Ani{}
	}

	var releaseDateList []string
	seenMonth := map[string]bool{}
	for i, ani := range sorted {
		if ani == nil {
			continue
		}
		ani.Sort = i
		if !ani.ReleaseDate.Time().IsZero() {
			month := ani.ReleaseDate.Time().Format("2006-01")
			if !seenMonth[month] {
				seenMonth[month] = true
				releaseDateList = append(releaseDateList, month)
			}
			wd := int(ani.ReleaseDate.Time().Weekday())
			weekItems[weeks[wd]] = append(weekItems[weeks[wd]], ani)
		}
	}
	sort.SliceStable(releaseDateList, func(i, j int) bool { return releaseDateList[i] > releaseDateList[j] })

	// week order: current weekday first then wrap
	today := int(model.Now().Weekday())
	order := make([]string, 0, 7)
	for i := today; i >= 0; i-- {
		order = append(order, weeks[i])
	}
	for i := 6; i > 0; i-- {
		if !containsStr(order, weeks[i]) {
			order = append(order, weeks[i])
		}
	}
	var weekList []model.WeekAni
	for _, w := range order {
		weekList = append(weekList, model.WeekAni{WeekLabel: w, Items: weekItems[w]})
	}
	return &model.ListAni{
		ReleaseDateList: releaseDateList,
		WeekList:        weekList,
		Total:           len(sorted),
	}
}

// AddAni adds a subscription.
func AddAni(ani *model.Ani) error {
	if ani == nil || strings.TrimSpace(ani.ID) == "" {
		ani = model.DefaultAni()
	}
	list := config.AniList()
	for _, a := range list {
		if a != nil && a.ID == ani.ID {
			return fmt.Errorf("订阅已存在")
		}
	}
	// duplicate title+season
	for _, a := range list {
		if a != nil && a.Title == ani.Title && a.Season == ani.Season {
			if config.Get().Replace {
				origID := a.ID
				*a = *ani
				a.ID = origID
				return config.SyncAni()
			}
			return fmt.Errorf("已存在同名订阅")
		}
	}
	list = append(list, ani)
	if err := config.SaveAniList(list); err != nil {
		return err
	}
	go func() {
		time.Sleep(time.Second)
		if !download.Login(true) {
			return
		}
		if ani.Enable {
			DownloadAni(ani)
		} else {
			items := rss.GetItems(ani)
			ani.CurrentEpisodeNumber = RssCurrentEpisodeNumber(ani, items)
			_ = config.SyncAni()
		}
	}()
	return nil
}

// SetAni updates a subscription (partial merge, preserving server-managed fields).
// SetAniRaw updates a subscription from raw JSON, copying only present fields.
// move=true also relocates the local files, download task save paths and the
// torrent cache (mirrors AniService.setAni with the `move` parameter).
func SetAniRaw(raw []byte, move bool) error {
	srcMap := map[string]interface{}{}
	if err := json.Unmarshal(raw, &srcMap); err != nil {
		return err
	}
	list := config.AniList()
	for i, a := range list {
		if a == nil {
			continue
		}
		var id string
		if v, ok := srcMap["id"].(string); ok {
			id = v
		}
		if a.ID != id {
			continue
		}
		title := strval(srcMap["title"])
		season := intval(srcMap["season"])
		// duplicate title+season check (excluding self)
		for j, other := range list {
			if other == nil || j == i {
				continue
			}
			if other.Title == title && other.Season == season {
				return fmt.Errorf("订阅标题重复")
			}
		}
		prev := a.Clone()
		MergeAniMap(a, srcMap)
		_ = config.SyncAni()

		if move {
			moveAniFiles(prev, a)
		} else if prev.CustomDownloadPathTemplate != a.CustomDownloadPathTemplate || prev.CustomDownloadPath != a.CustomDownloadPath {
			go func() {
				for _, t := range FindTorrentsInfosByAni(prev) {
					download.Type().SetSavePath(t, GetDownloadPath(a))
				}
			}()
		}
		return nil
	}
	return fmt.Errorf("订阅不存在")
}

// moveAniFiles relocates download tasks, local files and torrent cache when a
// subscription's download path changes (move=true).
func moveAniFiles(oldAni, newAni *model.Ani) {
	oldPath := GetDownloadPath(oldAni)
	newPath := GetDownloadPath(newAni)
	if oldPath == newPath {
		return
	}
	go func() {
		// repoint download task save paths
		for _, t := range FindTorrentsInfosByAni(oldAni) {
			download.Type().SetSavePath(t, newPath)
		}
		// move local files
		if err := os.MkdirAll(newPath, 0o755); err == nil {
			if entries, err := os.ReadDir(oldPath); err == nil {
				for _, e := range entries {
					_ = os.Rename(filepath.Join(oldPath, e.Name()), filepath.Join(newPath, e.Name()))
				}
			}
		}
		ClearDir(oldPath)
	}()
	// move torrent cache directory
	oldDir := TorrentDir(oldAni)
	newDir := TorrentDir(newAni)
	if oldDir != newDir {
		if err := os.MkdirAll(newDir, 0o755); err == nil {
			if entries, err := os.ReadDir(oldDir); err == nil {
				for _, e := range entries {
					_ = os.Rename(filepath.Join(oldDir, e.Name()), filepath.Join(newDir, e.Name()))
				}
			}
		}
		_ = os.RemoveAll(oldDir)
	}
}

// MergeAniMap copies JSON-present fields from srcMap onto dst, preserving
// currentEpisodeNumber and lastDownloadTime (BeanUtil.copyProperties semantics).
func MergeAniMap(dst *model.Ani, srcMap map[string]interface{}) {
	dstBytes, _ := json.Marshal(dst)
	dstMap := map[string]interface{}{}
	_ = json.Unmarshal(dstBytes, &dstMap)
	for k, v := range srcMap {
		if k == "currentEpisodeNumber" || k == "lastDownloadTime" {
			continue
		}
		dstMap[k] = v
	}
	mergedBytes, err := json.Marshal(dstMap)
	if err != nil {
		return
	}
	merged := &model.Ani{}
	if err := json.Unmarshal(mergedBytes, merged); err != nil {
		return
	}
	merged.CurrentEpisodeNumber = dst.CurrentEpisodeNumber
	merged.LastDownloadTime = dst.LastDownloadTime
	*dst = *merged
}

func strval(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func intval(v interface{}) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	}
	return 0
}

// DeleteAni removes subscriptions, optionally deleting files.
func DeleteAni(ids []string, deleteFiles bool) {
	list := config.AniList()
	var remaining []*model.Ani
	for _, a := range list {
		if a == nil {
			continue
		}
		if containsStr(ids, a.ID) {
			dir := TorrentDir(a)
			_ = os.RemoveAll(dir)
			if deleteFiles {
				path := GetDownloadPath(a)
				for _, t := range FindTorrentsInfosByAni(a) {
					DeleteTorrent(t, true, true)
				}
				_ = os.RemoveAll(path)
			}
			continue
		}
		remaining = append(remaining, a)
	}
	_ = config.SaveAniList(remaining)
}

// BatchEnable enables/disables subscriptions.
func BatchEnable(ids []string, value bool) {
	list := config.AniList()
	for _, a := range list {
		if a == nil || !containsStr(ids, a.ID) {
			continue
		}
		a.Enable = value
	}
	_ = config.SyncAni()
}

// RefreshAni triggers an async download round for one subscription.
func RefreshAni(ani *model.Ani) {
	go func() {
		time.Sleep(time.Second)
		if !download.Login(true) {
			log.Warnf("refresh", "%s 刷新失败: 115 未登录或 Cookie 无效", ani.Title)
			return
		}
		DownloadAni(ani)
	}()
}

// RefreshAll triggers an async download round for all enabled subscriptions.
func RefreshAll() {
	go func() {
		time.Sleep(time.Second)
		if !download.Login(true) {
			log.Warn("refresh", "刷新全部失败: 115 未登录或 Cookie 无效")
			return
		}
		for _, ani := range config.AniList() {
			if ani != nil && ani.Enable {
				DownloadAni(ani)
				time.Sleep(500 * time.Millisecond)
			}
		}
	}()
}

// UpdateTotalEpisodeNumber updates the total episode count from BGM.
func UpdateTotalEpisodeNumber(ids []string, force bool) {
	for _, ani := range config.AniList() {
		if ani == nil || (len(ids) > 0 && !containsStr(ids, ani.ID)) {
			continue
		}
		if !force && ani.TotalEpisodeNumber > 0 {
			continue
		}
		info, err := bgm.GetBgmInfo(GetSubjectIdFromAni(ani))
		if err == nil && info != nil && info.Eps > 0 {
			ani.TotalEpisodeNumber = info.Eps
		}
	}
	_ = config.SyncAni()
}

// GetSubjectIdFromAni resolves the bgm subject id via the rename hook.
func GetSubjectIdFromAni(ani *model.Ani) string {
	if rename.BgmSubjectId != nil {
		return rename.BgmSubjectId(ani)
	}
	return ""
}

// PreviewAni returns download path + items for the UI preview.
func PreviewAni(ani *model.Ani) map[string]interface{} {
	items := rss.GetItems(ani)
	savePath := GetDownloadPath(ani)
	var omitItems []int
	var preview []*model.Item
	for _, it := range items {
		it.HasDownloaded = ItemDownloaded(ani, it, true)
		preview = append(preview, it)
	}
	return map[string]interface{}{
		"downloadPath": savePath,
		"items":        preview,
		"omitList":     omitItems,
	}
}

// DownloadPathPreview returns the resolved download path for an ani.
func DownloadPathPreview(ani *model.Ani) map[string]interface{} {
	return map[string]interface{}{
		"downloadPath": GetDownloadPath(ani),
	}
}

// Completed migrates a finished subscription to the completed path.
func Completed(ani *model.Ani) {
	cfg := config.Get()
	if !ani.Completed || !cfg.AutoDisabled || !cfg.Completed || ani.Enable || ani.TotalEpisodeNumber < 1 || ani.CurrentEpisodeNumber < ani.TotalEpisodeNumber || ani.Ova {
		return
	}
	newPath := GetDownloadPathTemplate(ani, cfg.CompletedPathTemplate)
	oldPath := GetDownloadPath(ani)
	if newPath == oldPath {
		return
	}
	for _, t := range FindTorrentsInfosByAni(ani) {
		download.Type().SetSavePath(t, newPath)
	}
	moveLocalFiles(oldPath, newPath)
}

func moveLocalFiles(from, to string) {
	entries, err := os.ReadDir(from)
	if err != nil {
		return
	}
	if err := os.MkdirAll(to, 0o755); err != nil {
		return
	}
	for _, e := range entries {
		_ = os.Rename(filepath.Join(from, e.Name()), filepath.Join(to, e.Name()))
	}
}