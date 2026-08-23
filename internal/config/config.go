package config

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"ani-rss/internal/model"
	"ani-rss/internal/store"
)

var (
	mu     sync.RWMutex
	cfg    *model.Config
	aniLst []*model.Ani
	dir    string
)

// Paths to the persisted JSON files.
const (
	ConfigFile = "config.v2.json"
	AniFile    = "ani.v2.json"
)

// Dir returns the resolved config directory (absolute).
func Dir() string { return dir }

// ConfigDirFile returns an absolute path under the config dir.
func ConfigDirFile(rel string) string { return filepath.Join(dir, rel) }

// Get returns the global Config singleton (never nil).
func Get() *model.Config {
	mu.RLock()
	defer mu.RUnlock()
	return cfg
}

// Sync persists the current config to disk.
func Sync() error {
	mu.RLock()
	c := cfg
	mu.RUnlock()
	return store.WriteJSON(ConfigDirFile(ConfigFile), c)
}

// AniList returns the current subscription list.
func AniList() []*model.Ani {
	mu.RLock()
	defer mu.RUnlock()
	return aniLst
}

// SaveAniList persists the subscription list and updates the in-memory copy.
func SaveAniList(list []*model.Ani) error {
	mu.Lock()
	aniLst = list
	mu.Unlock()
	return SyncAni()
}

// SyncAni persists the current in-memory subscription list.
func SyncAni() error {
	mu.RLock()
	l := aniLst
	mu.RUnlock()
	return store.WriteJSON(ConfigDirFile(AniFile), l)
}

// Load resolves the config dir and reads both config and subscription files.
// It creates them with defaults if missing.
func Load() error {
	d, err := resolveDir()
	if err != nil {
		return err
	}
	dir = d

	defCfg := model.DefaultConfig()
	if defCfg.UUID == "" {
		defCfg.UUID = randomHex(16)
	}
	cfgPath := ConfigDirFile(ConfigFile)
	// write defaults when the config file does not exist yet
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		if err := store.WriteJSON(cfgPath, defCfg); err != nil {
			return err
		}
	}
	c := &model.Config{}
	if err := store.ReadJSON(cfgPath, c); err != nil {
		return err
	}
	// record which keys are present so defaults apply only to missing fields
	// (mirrors Java: defaults + file overrides)
	present := map[string]bool{}
	if raw, err := os.ReadFile(cfgPath); err == nil {
		var rm map[string]json.RawMessage
		if json.Unmarshal(raw, &rm) == nil {
			for k := range rm {
				present[k] = true
			}
		}
	}
	fillConfigDefaults(c, defCfg, present)
	if c.UUID == "" {
		c.UUID = randomHex(16)
	}
	cfg = c

	var list []*model.Ani
	if err := store.ReadJSON(ConfigDirFile(AniFile), &list); err != nil {
		return err
	}
	for _, a := range list {
		if a == nil {
			continue
		}
		fillAniDefaults(a)
	}
	aniLst = list
	return nil
}

// MergeConfigInto merges only the JSON-present fields from raw into cur
// (mirroring the Java BeanUtil.copyProperties(ignoreNullValue) behavior for
// partial config POSTs). Blank login credentials preserve the stored ones.
func MergeConfigInto(cur *model.Config, raw []byte) error {
	curBytes, err := json.Marshal(cur)
	if err != nil {
		return err
	}
	curMap := map[string]interface{}{}
	if err := json.Unmarshal(curBytes, &curMap); err != nil {
		return err
	}
	incoming := map[string]interface{}{}
	if err := json.Unmarshal(raw, &incoming); err != nil {
		return err
	}
	for k, v := range incoming {
		if k == "gitInfo" {
			continue
		}
		curMap[k] = v
	}
	mergedBytes, err := json.Marshal(curMap)
	if err != nil {
		return err
	}
	merged := &model.Config{}
	if err := json.Unmarshal(mergedBytes, merged); err != nil {
		return err
	}
	// preserve blank login credentials
	if merged.Login.Username == "" {
		merged.Login.Username = cur.Login.Username
	}
	if merged.Login.Password == "" {
		merged.Login.Password = cur.Login.Password
	}
	merged.UUID = cur.UUID
	merged.GitInfo = cur.GitInfo
	*cur = *merged
	return nil
}

// fillConfigDefaults applies defaults for unset fields, honoring Java's
// "defaults + file overrides" semantics: a default only applies when the key
// was absent from the JSON file (present holds the file's keys).
func fillConfigDefaults(c, def *model.Config, present map[string]bool) {
	if !present["mikanHost"] {
		c.MikanHost = def.MikanHost
	}
	if !present["tmdbApi"] {
		c.TmdbApi = def.TmdbApi
	}
	if !present["tmdbApiKey"] {
		c.TmdbApiKey = def.TmdbApiKey
	}
	if !present["tmdbAnime"] {
		c.TmdbAnime = def.TmdbAnime
	}
	if !present["downloadToolType"] {
		c.DownloadToolType = def.DownloadToolType
	}
	if !present["downloadRetry"] {
		c.DownloadRetry = def.DownloadRetry
	}
	if !present["downloadPathTemplate"] {
		c.DownloadPathTemplate = def.DownloadPathTemplate
	}
	if !present["ovaDownloadPathTemplate"] {
		c.OvaDownloadPathTemplate = def.OvaDownloadPathTemplate
	}
	if !present["rssSleepMinutes"] {
		c.RssSleepMinutes = def.RssSleepMinutes
	}
	if !present["renameSleepSeconds"] {
		c.RenameSleepSeconds = def.RenameSleepSeconds
	}
	if !present["rename"] {
		c.Rename = def.Rename
	}
	if !present["rss"] {
		c.Rss = def.Rss
	}
	if !present["rssTimeout"] {
		c.RssTimeout = def.RssTimeout
	}
	if !present["awaitStalledUP"] {
		c.AwaitStalledUP = def.AwaitStalledUP
	}
	if !present["titleYear"] {
		c.TitleYear = def.TitleYear
	}
	if !present["skip5"] {
		c.Skip5 = def.Skip5
	}
	if !present["logsMax"] {
		c.LogsMax = def.LogsMax
	}
	if !present["procrastinatingMasterOnly"] {
		c.ProcrastinatingMasterOnly = def.ProcrastinatingMasterOnly
	}
	if !present["proxyPort"] {
		c.ProxyPort = def.ProxyPort
	}
	if !present["login"] {
		c.Login = def.Login
	}
	if !present["loginEffectiveHours"] {
		c.LoginEffectiveHours = def.LoginEffectiveHours
	}
	if !present["multiLoginForbidden"] {
		c.MultiLoginForbidden = def.MultiLoginForbidden
	}
	if !present["exclude"] {
		c.Exclude = def.Exclude
	}
	if !present["tmdb"] {
		c.Tmdb = def.Tmdb
	}
	if !present["tmdbLanguage"] {
		c.TmdbLanguage = def.TmdbLanguage
	}
	if !present["omit"] {
		c.Omit = def.Omit
	}
	if !present["customEpisodeGroupIndex"] {
		c.CustomEpisodeGroupIndex = def.CustomEpisodeGroupIndex
	}
	if !present["customEpisodeStr"] {
		c.CustomEpisodeStr = def.CustomEpisodeStr
	}
	if !present["procrastinatingDay"] {
		c.ProcrastinatingDay = def.ProcrastinatingDay
	}
	if !present["openListDownloadTimeout"] {
		c.OpenListDownloadTimeout = def.OpenListDownloadTimeout
	}
	if !present["openListDownloadRetryNumber"] {
		c.OpenListDownloadRetryNumber = def.OpenListDownloadRetryNumber
	}
	if !present["configBackupDay"] {
		c.ConfigBackupDay = def.ConfigBackupDay
	}
	if !present["sortType"] {
		c.SortType = def.SortType
	}
	if !present["followDay"] {
		c.FollowDay = def.FollowDay
	}
	if !present["limitLoginAttempts"] {
		c.LimitLoginAttempts = def.LimitLoginAttempts
	}
	if !present["bgmApi"] {
		c.BgmApi = def.BgmApi
	}
	if !present["subtitleIndependentFolderName"] {
		c.SubtitleIndependentFolderName = def.SubtitleIndependentFolderName
	}
	if !present["notificationTemplate"] {
		c.NotificationTemplate = def.NotificationTemplate
	}
	if !present["bgmImage"] {
		c.BgmImage = def.BgmImage
	}
	if !present["customCss"] {
		c.CustomCss = def.CustomCss
	}
	if !present["customJs"] {
		c.CustomJs = def.CustomJs
	}
	if !present["gitInfo"] {
		c.GitInfo = def.GitInfo
	}
	// always serialize lists as [] (frontend relies on .length/.push)
	if c.Exclude == nil {
		c.Exclude = []string{}
	}
	if c.CustomTags == nil {
		c.CustomTags = []string{}
	}
	if c.PriorityKeywords == nil {
		c.PriorityKeywords = []string{}
	}
	if c.NotificationConfigList == nil {
		c.NotificationConfigList = []model.NotificationConfig{}
	}
	if c.ReverseProxyTrustIpList == nil {
		c.ReverseProxyTrustIpList = []string{}
	}
}

func fillAniDefaults(a *model.Ani) {
	if len(a.Exclude) == 0 {
		a.Exclude = []string{"720[Pp]", "\\d-\\d", "合集", "特别篇"}
	}
	if a.CustomEpisodeStr == "" {
		a.CustomEpisodeStr = model.RENAME_REG_STR()
	}
	if a.CustomEpisodeGroupIndex == 0 {
		a.CustomEpisodeGroupIndex = 2
	}
	if a.CustomRenameTemplate == "" {
		a.CustomRenameTemplate = "[${subgroup}] ${title} S${seasonFormat}E${episodeFormat}"
	}
	// always serialize lists as [] (frontend relies on .length/.push)
	if a.StandbyRssList == nil {
		a.StandbyRssList = []model.StandbyRss{}
	}
	if a.Match == nil {
		a.Match = []string{}
	}
	if a.NotDownload == nil {
		a.NotDownload = []float64{}
	}
	if a.CustomPriorityKeywords == nil {
		a.CustomPriorityKeywords = []string{}
	}
	if a.CustomTags == nil {
		a.CustomTags = []string{}
	}
}

// resolveDir mirrors ConfigUtil.getConfigDir():
//  1. env CONFIG or CLI --config
//  2. ./config if it exists
//  3. ~/ani-rss on Windows/macOS, ./config on Linux
func resolveDir() (string, error) {
	if d := os.Getenv("CONFIG"); d != "" {
		abs, err := filepath.Abs(d)
		if err != nil {
			return "", err
		}
		return abs, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	local := filepath.Join(wd, "config")
	if _, err := os.Stat(local); err == nil {
		return local, nil
	}
	return local, nil
}

func randomHex(n int) string {
	b := make([]byte, n)
	cryptRandFill(b)
	return hex.EncodeToString(b)
}

var cryptRandFill = func(b []byte) {
	cr, err := os.Open("/dev/urandom")
	if err != nil {
		panic(err)
	}
	defer cr.Close()
	if _, err := cr.Read(b); err != nil {
		panic(err)
	}
}

// CleanPath normalizes slash handling (mirrors path normalization).
func CleanPath(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	p = strings.TrimSuffix(p, "/")
	return p
}