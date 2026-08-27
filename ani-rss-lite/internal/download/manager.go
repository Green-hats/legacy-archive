package download

import (
	"strings"
	"sync"

	"ani-rss/internal/config"
	"ani-rss/internal/model"
)

var (
	mu      sync.RWMutex
	current Client

	loginStatusMu sync.Mutex
	loginStatus   LoginStatus
)

// LoginStatus describes the latest download-client login result.
type LoginStatus struct {
	Configured bool   `json:"configured"`
	OK         bool   `json:"loginOK"`
	Message    string `json:"message"`
}

// SetLoginStatus records the latest client login result.
func SetLoginStatus(s LoginStatus) {
	loginStatusMu.Lock()
	loginStatus = s
	loginStatusMu.Unlock()
}

// GetLoginStatus returns the latest client login result.
func GetLoginStatus() LoginStatus {
	loginStatusMu.Lock()
	defer loginStatusMu.Unlock()
	return loginStatus
}

// Type returns the normalized download client for the configured tool type.
// When the config tool type changes the client is rebuilt lazily.
func Type() Client {
	cfg := config.Get()
	mu.RLock()
	if current != nil {
		mu.RUnlock()
		return current
	}
	mu.RUnlock()
	mu.Lock()
	defer mu.Unlock()
	if current == nil {
		current = build(cfg.DownloadToolType)
	}
	return current
}

// Reload rebuilds the client (called when downloadToolType changes).
func Reload() {
	mu.Lock()
	defer mu.Unlock()
	current = nil
	ResetWarnings()
}

func build(t string) Client {
	switch strings.ToLower(t) {
	case "115", "pan115", "115网盘":
		return &Pan115{}
	default:
		return &PikPak{}
	}
}

// Login attempts to log into the configured download client.
func Login(test bool) bool {
	return Type().Login(test, config.Get())
}

// GetTorrentsInfos returns the client torrent snapshot.
func GetTorrentsInfos() []*model.TorrentsInfo {
	return Type().GetTorrentsInfos()
}

// FindByHash looks up a torrent by hash.
func FindByHash(hash string) *model.TorrentsInfo {
	for _, t := range GetTorrentsInfos() {
		if strings.EqualFold(t.Hash, hash) {
			return t
		}
	}
	return nil
}

// SetFindAniHook registers the service-side resolver used by rename flows.
func SetFindAniHook(fn func(t *model.TorrentsInfo) *model.Ani) {
	FindAniByDownloadPath = fn
}