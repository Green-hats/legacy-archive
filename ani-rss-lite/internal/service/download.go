package service

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"ani-rss/internal/cache"
	"ani-rss/internal/config"
	"ani-rss/internal/download"
	"ani-rss/internal/log"
	"ani-rss/internal/model"
	"ani-rss/internal/notify"
	"ani-rss/internal/rename"
	"ani-rss/internal/rss"
	"ani-rss/internal/util"
)

var (
	regSeasonEp   = regexp.MustCompile(`[Ss](\d+)[Ee](\d+(\.5)?)`)
	downloadMutex = make(chan struct{}, 1)
)

// SeasonEpisodeRe exposes the SxxExx regex for other packages.
var SeasonEpisodeRe = regSeasonEp

// GetDownloadPath resolves the download path template for an ani.
func GetDownloadPath(ani *model.Ani) string {
	return getDownloadPath(ani, config.Get())
}

// GetDownloadPathTemplate resolves using a specific template.
func GetDownloadPathTemplate(ani *model.Ani, tmpl string) string {
	clone := ani.Clone()
	clone.CustomDownloadPathTemplate = tmpl
	clone.CustomDownloadPath = true
	return getDownloadPath(clone, config.Get())
}

func getDownloadPath(ani *model.Ani, cfg *model.Config) string {
	tmpl := cfg.DownloadPathTemplate
	if ani.Ova && cfg.OvaDownloadPathTemplate != "" {
		tmpl = cfg.OvaDownloadPathTemplate
	}
	if ani.CustomDownloadPath && strings.TrimSpace(ani.CustomDownloadPathTemplate) != "" {
		for _, line := range strings.Split(ani.CustomDownloadPathTemplate, "\n") {
			if line = strings.TrimSpace(line); line != "" {
				tmpl = line
				break
			}
		}
	}

	title := strings.TrimSpace(ani.Title)
	letter := util.GetPinyinInitials(title)
	tmpl = strings.ReplaceAll(tmpl, "${letter}", letter)

	releaseDate := ani.ReleaseDate.Time()
	if releaseDate.IsZero() {
		releaseDate = time.Now()
	}
	tmdbDate := releaseDate
	if ani.Tmdb != nil && !ani.Tmdb.Date.Time().IsZero() {
		tmdbDate = ani.Tmdb.Date.Time()
	}
	tmdbYear := tmdbDate.Year()
	year := releaseDate.Year()
	month := int(releaseDate.Month())
	monthFormat := fmt.Sprintf("%02d", month)

	if strings.Contains(tmpl, "${quarter}") || strings.Contains(tmpl, "${quarterFormat}") || strings.Contains(tmpl, "${quarterName}") {
		var quarter int
		var quarterName string
		switch month {
		case 12, 1, 2:
			if month == 12 {
				year++
			}
			quarter = 1
			quarterName = "冬"
		case 3, 4, 5:
			quarter = 4
			quarterName = "春"
		case 6, 7, 8:
			quarter = 7
			quarterName = "夏"
		default:
			quarter = 10
			quarterName = "秋"
		}
		tmpl = strings.ReplaceAll(tmpl, "${quarter}", strconv.Itoa(quarter))
		tmpl = strings.ReplaceAll(tmpl, "${quarterFormat}", fmt.Sprintf("%02d", quarter))
		tmpl = strings.ReplaceAll(tmpl, "${quarterName}", quarterName)
	}

	tmpl = strings.ReplaceAll(tmpl, "${tmdbYear}", strconv.Itoa(tmdbYear))
	tmpl = strings.ReplaceAll(tmpl, "${year}", strconv.Itoa(year))
	tmpl = strings.ReplaceAll(tmpl, "${month}", strconv.Itoa(month))
	tmpl = strings.ReplaceAll(tmpl, "${monthFormat}", monthFormat)

	season := ani.Season
	tmpl = strings.ReplaceAll(tmpl, "${season}", strconv.Itoa(season))
	tmpl = strings.ReplaceAll(tmpl, "${seasonFormat}", fmt.Sprintf("%02d", season))

	if strings.Contains(tmpl, "${bgmId}") {
		if rename.BgmSubjectId != nil {
			tmpl = strings.ReplaceAll(tmpl, "${bgmId}", rename.BgmSubjectId(ani))
		} else {
			tmpl = strings.ReplaceAll(tmpl, "${bgmId}", "")
		}
	}
	tmpl = strings.ReplaceAll(tmpl, "${title}", ani.Title)
	tmpl = strings.ReplaceAll(tmpl, "${themoviedbName}", ani.ThemoviedbName)
	tmpl = strings.ReplaceAll(tmpl, "${subgroup}", ani.Subgroup)

	tmdbId := ""
	if ani.Tmdb != nil && ani.Tmdb.ID != 0 {
		tmdbId = strconv.Itoa(ani.Tmdb.ID)
	}
	tmpl = strings.ReplaceAll(tmpl, "${tmdbid}", tmdbId)

	if strings.Contains(tmpl, "${jpTitle}") {
		jp := ani.JpTitle
		if jp == "" && rename.JpTitle != nil {
			jp = rename.JpTitle(ani)
		}
		tmpl = strings.ReplaceAll(tmpl, "${jpTitle}", jp)
	}

	return config.CleanPath(tmpl)
}

// FindAniByDownloadPath finds the subscription owning a torrent save path.
func FindAniByDownloadPath(t *model.TorrentsInfo) *model.Ani {
	dir := t.SavePath
	for _, ani := range config.AniList() {
		if ani == nil {
			continue
		}
		if GetDownloadPath(ani) == dir {
			return ani.Clone()
		}
	}
	return nil
}

// FindTorrentsInfosByAni returns the client torrents whose save path matches the ani.
func FindTorrentsInfosByAni(ani *model.Ani) []*model.TorrentsInfo {
	path := GetDownloadPath(ani)
	var out []*model.TorrentsInfo
	for _, t := range download.GetTorrentsInfos() {
		if t.SavePath == path {
			out = append(out, t)
		}
	}
	return out
}

// DownloadAni is the core per-subscription download round.
func DownloadAni(ani *model.Ani) {
	downloadMutex <- struct{}{}
	defer func() { <-downloadMutex }()

	cfg := config.Get()
	title := ani.Title
	items := rss.GetItems(ani)
	RssOmit(ani, items)
	log.Infof("download", "%s 刷新完成, 共 %d 个条目", title, len(items))

	torrentsInfos := download.GetTorrentsInfos()

	savePath := GetDownloadPath(ani)
	RssProcrastinating(ani, items)

	sync := false
	currentDownloadCount := 0

	for _, item := range items {
		reName := item.ReName
		torrentPath := TorrentFile(ani, item)
		master := item.Master
		hash := strings.ToLower(strings.TrimSuffix(filepath.Base(torrentPath), filepath.Ext(torrentPath)))
		episode := item.Episode
		is5 := rename.Is5(episode)

		if fi, err := os.Stat(torrentPath); err == nil && fi.Size() >= 0 {
			log.Debugf("download", "种子记录已存在 %s", reName)
			if master && !is5 {
				currentDownloadCount++
			}
			continue
		}

		if containsFloat(ani.NotDownload, episode) {
			if master && !is5 {
				currentDownloadCount++
			}
			continue
		}

		if ani.DownloadNew {
			newItem := items[len(items)-1]
			if !item.PubDate.Time().IsZero() && !newItem.PubDate.Time().IsZero() {
				if item.PubDate.Time().Format("2006-01-02") != newItem.PubDate.Time().Format("2006-01-02") {
					if master && !is5 {
						currentDownloadCount++
					}
					continue
				}
			} else if item != newItem {
				if master && !is5 {
					currentDownloadCount++
				}
				continue
			}
		}

		if !item.PubDate.Time().IsZero() && cfg.DelayedDownload > 0 {
			if time.Now().Add(-time.Duration(cfg.DelayedDownload) * time.Minute).Before(item.PubDate.Time()) {
				log.Infof("download", "延迟下载 %s", reName)
				continue
			}
		}

		dup := false
		for _, t := range torrentsInfos {
			if strings.EqualFold(t.Hash, hash) {
				dup = true
				break
			}
		}
		if dup {
			log.Infof("download", "已有下载任务 hash:%s name:%s", hash, reName)
			if master && !is5 {
				currentDownloadCount++
			}
			continue
		}

		if ItemDownloaded(ani, item, true) {
			log.Infof("download", "本地文件已存在 %s", reName)
			if master && !is5 {
				currentDownloadCount++
			}
			continue
		}

		saveTorrent := SaveTorrent(ani, item)
		if fi, err := os.Stat(saveTorrent); err != nil || fi.Size() < 0 {
			continue
		}

		sync = true
		downloadAni(ani, item, savePath, saveTorrent)
		if master && !is5 {
			currentDownloadCount++
		}
	}

	if sync {
		ani.CurrentEpisodeNumber = RssCurrentEpisodeNumber(ani, items)
		ani.LastDownloadTime = model.Now().UnixMilli()
		_ = config.SyncAni()
	}

	if !cfg.AutoDisabled {
		return
	}
	if ani.TotalEpisodeNumber < 1 {
		return
	}
	if currentDownloadCount >= ani.TotalEpisodeNumber {
		log.Infof("download", "%s 第 %d 季 共 %d 集 已全部下载完成, 自动停止订阅", title, ani.Season, ani.TotalEpisodeNumber)
		notify.Send(cfg, ani, fmt.Sprintf("%s 订阅已完结", title), model.NotifyCompleted)
		ani.Enable = false
		_ = config.SyncAni()
	}
}

func containsFloat(list []float64, v float64) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func containsStr(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// downloadAni performs the actual client download with retries.
func downloadAni(ani *model.Ani, item *model.Item, savePath, torrentPath string) {
	cfg := config.Get()
	clone := ani.Clone()
	subgroup := item.Subgroup
	if strings.TrimSpace(subgroup) == "" {
		subgroup = "未知字幕组"
	}
	clone.Subgroup = subgroup

	name := item.ReName
	log.Infof("download", "添加下载 %s", name)

	if _, err := os.Stat(torrentPath); err != nil {
		log.Errorf("download", "种子下载出现问题 %s %s", name, torrentPath)
		return
	}
	time.Sleep(time.Second)
	savePath = config.CleanPath(savePath)

	text := fmt.Sprintf("%s 已更新", name)
	if !item.Master {
		text = fmt.Sprintf("(备用RSS) %s", text)
	}
	notify.Send(cfg, clone, text, model.NotifyDownloadStart)

	retry := cfg.DownloadRetry
	if retry <= 0 {
		retry = 3
	}
	ok := false
	for i := 1; i <= retry; i++ {
		if download.Type().Download(clone, item, savePath, torrentPath) {
			ok = true
			break
		}
		log.Errorf("download", "%s 下载失败将进行重试, 当前重试次数为%d次", name, i)
	}
	if !ok {
		_ = os.Remove(torrentPath)
		log.Errorf("download", "%s 添加失败，疑似为坏种", name)
		notify.Send(cfg, clone, fmt.Sprintf("%s 添加失败，疑似为坏种", name), model.NotifyError)
	}
}

// ItemDownloaded checks whether a local file already matches the item.
func ItemDownloaded(ani *model.Ani, item *model.Item, checkDownloadList bool) bool {
	cfg := config.Get()
	if !cfg.Rename || !cfg.FileExist || cfg.DownloadPathTemplate == "" {
		return false
	}
	season := ani.Season
	reName := item.ReName
	episode := item.Episode

	if checkDownloadList {
		for _, t := range FindTorrentsInfosByAni(ani) {
			if !strings.EqualFold(t.Name, reName) {
				continue
			}
			log.Infof("download", "已存在下载任务 %s", reName)
			SaveTorrent(ani, item)
			return true
		}
	}

	downloadPath := GetDownloadPath(ani)
	// cloud downloaders (PikPak / OpenList) store files remotely
	if cloud, ok := download.Type().(download.CloudClient); ok {
		if cloud.FileExists(downloadPath + "/" + reName) {
			SaveTorrent(ani, item)
			log.Infof("download", "云端已存在 %s", reName)
			return true
		}
		return false
	}
	entries, err := os.ReadDir(downloadPath)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() || !util.IsVideo(e.Name()) {
			continue
		}
		if ani.Ova {
			return true
		}
		m := regSeasonEp.FindStringSubmatch(e.Name())
		if len(m) < 3 {
			continue
		}
		seasonStr, err1 := strconv.Atoi(m[1])
		episodeStr, err2 := strconv.ParseFloat(m[2], 64)
		if err1 != nil || err2 != nil {
			continue
		}
		if season == seasonStr && episode == episodeStr {
			SaveTorrent(ani, item)
			log.Infof("download", "本地已存在 %s", reName)
			return true
		}
	}
	return false
}

// DeleteTorrent removes a torrent from the client (mirrors TorrentUtil.delete).
func DeleteTorrent(t *model.TorrentsInfo, forcedDelete, deleteFiles bool) bool {
	if !forcedDelete {
		if !AllowDelete(t) {
			return false
		}
	}
	log.Infof("download", "删除任务 title: %s forcedDelete: %v deleteFiles: %v", t.Name, forcedDelete, deleteFiles)
	time.Sleep(500 * time.Millisecond)
	if !download.Type().Delete(t, deleteFiles) {
		log.Errorf("download", "删除任务失败 %s", t.Name)
		return false
	}
	log.Infof("download", "删除任务成功 %s", t.Name)
	if deleteFiles {
		ClearDir(t.SavePath)
	}
	return true
}

// ClearDir removes empty directories under a path.
func ClearDir(path string) {
	_ = filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() || p == path {
			return nil
		}
		entries, err := os.ReadDir(p)
		if err != nil {
			return nil
		}
		if len(entries) == 0 {
			_ = os.Remove(p)
		}
		return nil
	})
}

// RssOmit notifies about missing episodes.
func RssOmit(ani *model.Ani, items []*model.Item) {
	cfg := config.Get()
	if !cfg.Omit || !ani.Omit || ani.Ova || len(items) == 0 {
		return
	}
	distinct := map[int]bool{}
	for _, it := range items {
		distinct[int(it.Episode)] = true
	}
	eps := make([]int, 0, len(distinct))
	for e := range distinct {
		eps = append(eps, e)
	}
	sort.Ints(eps)
	if len(eps) == 0 {
		return
	}
	min, max := eps[0], eps[len(eps)-1]
	if min == max {
		return
	}
	var missing []int
	for ep := min; ep <= max; ep++ {
		if distinct[ep] {
			continue
		}
		if len(missing) > 50 {
			return
		}
		missing = append(missing, ep)
	}
	if len(missing) == 0 || len(missing) > 10 {
		return
	}
	var sList []string
	for _, ep := range missing {
		s := fmt.Sprintf("缺少集数 %s S%02dE%02d", ani.Title, ani.Season, ep)
		key := fmt.Sprintf("omit:%s:ep-%d", ani.ID, ep)
		if cache.Default.Contains(key) {
			continue
		}
		log.Info("download", s)
		cache.Default.PutDuration(key, s, 24*time.Hour)
		sList = append(sList, s)
	}
	if len(sList) > 0 {
		notify.Send(cfg, ani, strings.Join(sList, "\n"), model.NotifyOmit)
	}
}

// RssProcrastinating notifies when the latest release is old.
func RssProcrastinating(ani *model.Ani, items []*model.Item) {
	cfg := config.Get()
	if !cfg.Procrastinating || !ani.Procrastinating {
		return
	}
	if cfg.ProcrastinatingMasterOnly {
		var master []*model.Item
		for _, it := range items {
			if it.Master {
				master = append(master, it)
			}
		}
		items = master
	}
	var latest time.Time
	for _, it := range items {
		if !it.PubDate.Time().IsZero() && it.PubDate.Time().After(latest) {
			latest = it.PubDate.Time()
		}
	}
	if latest.IsZero() || latest.After(time.Now()) {
		return
	}
	day := int(time.Since(latest).Hours() / 24)
	if day < cfg.ProcrastinatingDay {
		return
	}
	text := fmt.Sprintf("检测到%s, 已摸鱼%d天", ani.Title, day)
	key := "procrastinating:" + ani.ID
	if cache.Default.Contains(key) {
		return
	}
	cache.Default.PutDuration(key, text, 24*time.Hour)
	notify.Send(cfg, ani, text, model.NotifyProcrastinating)
}

// RssCurrentEpisodeNumber computes the current episode count.
func RssCurrentEpisodeNumber(ani *model.Ani, items []*model.Item) int {
	cfg := config.Get()
	if cfg.StandbyRss && cfg.Coexist {
		var master []*model.Item
		for _, it := range items {
			if it.Master {
				master = append(master, it)
			}
		}
		items = master
	}
	var cleaned []*model.Item
	for _, it := range items {
		if it.Episode == float64(int(it.Episode)) {
			cleaned = append(cleaned, it)
		}
	}
	if len(cleaned) == 0 {
		return 0
	}
	if ani.DownloadNew {
		max := 0
		for _, it := range cleaned {
			if int(it.Episode) > max {
				max = int(it.Episode)
			}
		}
		return max
	}
	return len(cleaned)
}

