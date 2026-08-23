package task

import (
	"sync"
	"sync/atomic"
	"time"

	"ani-rss/internal/bgm"
	"ani-rss/internal/config"
	"ani-rss/internal/download"
	"ani-rss/internal/log"
	"ani-rss/internal/model"
	"ani-rss/internal/service"
)

var (
	mu      sync.Mutex
	running atomic.Bool
	stopCh  chan struct{}
)

// Start launches the three background loops: rename-task, rss-task, bgm-task.
func Start() {
	mu.Lock()
	defer mu.Unlock()
	if running.Load() {
		return
	}
	running.Store(true)
	stopCh = make(chan struct{})
	go loop("rename-task", stopCh, RenameLoop)
	go loop("rss-task", stopCh, RssLoop)
	go loop("bgm-task", stopCh, BgmLoop)
}

// Stop halts all background loops.
func Stop() {
	mu.Lock()
	defer mu.Unlock()
	if !running.Load() {
		return
	}
	running.Store(false)
	close(stopCh)
}

// Restart stops and starts the loops (after config timing changes).
func Restart() {
	Stop()
	Start()
}

func loop(name string, stop <-chan struct{}, fn func(stop <-chan struct{})) {
	for {
		select {
		case <-stop:
			return
		default:
			fn(stop)
		}
	}
}

// RssLoop polls RSS on an interval.
func RssLoop(stop <-chan struct{}) {
	cfg := config.Get()
	if !cfg.Rss {
		sleepMinutes(stop, cfg.RssSleepMinutes)
		return
	}
	SyncDownload(config.AniList())
	sleepMinutes(stop, cfg.RssSleepMinutes)
}

// SyncDownload runs one download round over all subscriptions.
func SyncDownload(list []*model.Ani) {
	if !download.Login(true) {
		log.Warn("rss-task", "下载客户端登录失败, 跳过本轮")
		return
	}
	for _, ani := range list {
		if ani == nil || !ani.Enable {
			continue
		}
		service.DownloadAni(ani)
		time.Sleep(500 * time.Millisecond)
	}
}

// RenameLoop processes completion: rename, notify, delete.
func RenameLoop(stop <-chan struct{}) {
	cfg := config.Get()
	if !download.Login(true) {
		sleepSeconds(stop, cfg.RenameSleepSeconds)
		return
	}
	for _, t := range download.GetTorrentsInfos() {
		if t == nil {
			continue
		}
		RenameAndNotify(t)
	}
	sleepSeconds(stop, cfg.RenameSleepSeconds)
}

// RenameAndNotify renames a torrent and triggers completion hooks.
func RenameAndNotify(t *model.TorrentsInfo) {
	cfg := config.Get()
	if cfg.Rename && !t.HasTag(model.TagRename) {
		if download.Type().Rename(t) {
			download.Type().AddTags(t, model.TagRename)
		}
	}
	service.Notification(t)
	if !cfg.DeleteStandbyRSSOnly {
		service.DeleteTorrent(t, false, false)
	}
}

// BgmLoop refreshes scores every 12 hours.
func BgmLoop(stop <-chan struct{}) {
	if err := bgm.RefreshToken(); err != nil {
		log.Debugf("bgm-task", "bgm token 刷新失败 %v", err)
	}
	cfg := config.Get()
	for _, ani := range config.AniList() {
		if ani == nil || !ani.Enable {
			continue
		}
		if info, err := bgm.GetBgmInfo(service.GetSubjectIdFromAni(ani)); err == nil && info != nil {
			ani.Score = info.Rating.Score
		}
		if cfg.UpdateTotalEpisodeNumber {
			service.UpdateTotalEpisodeNumber([]string{ani.ID}, cfg.ForceUpdateTotalEpisodeNumber)
		}
	}
	_ = config.SyncAni()
	sleepHours(stop, 12)
}

func sleepMinutes(stop <-chan struct{}, minutes int) {
	if minutes <= 0 {
		minutes = 15
	}
	sleepDuration(stop, time.Duration(minutes)*time.Minute)
}

func sleepSeconds(stop <-chan struct{}, seconds int) {
	if seconds <= 0 {
		seconds = 10
	}
	sleepDuration(stop, time.Duration(seconds)*time.Second)
}

func sleepHours(stop <-chan struct{}, hours int) {
	sleepDuration(stop, time.Duration(hours)*time.Hour)
}

func sleepDuration(stop <-chan struct{}, d time.Duration) {
	select {
	case <-stop:
	case <-time.After(d):
	}
}