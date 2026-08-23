package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ani-rss/internal/cache"
	"ani-rss/internal/config"
	"ani-rss/internal/model"
	"ani-rss/internal/util"
)

// GithubRelease mirrors the GitHub releases/latest payload fields we need.
type GithubRelease struct {
	TagName     string `json:"tag_name"`
	PublishedAt string `json:"published_at"`
	Body        string `json:"body"`
	Message     string `json:"message"`
	Assets      []struct {
		Name              string `json:"name"`
		Size              int64  `json:"size"`
		Digest            string `json:"digest"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// githubRepo is the repository used for update checks. 二创版默认关闭更新检查,
// 发布到自己的 GitHub 后改成 `yourname/ani-rss-go` 即可启用。
const githubRepo = ""

// CheckUpdate fetches the latest release info (mirrors UpdateService.about()).
func CheckUpdate() *model.About {
	cfg := config.Get()
	if v, ok := cache.Default.Get("github#releases-latest"); ok {
		if a, ok := v.(*model.About); ok {
			return a
		}
	}
	about := &model.About{
		Version:    util.Version,
		AutoUpdate: cfg.AutoUpdate,
		Date:       model.DateTime(model.Now()),
	}
	// 未配置更新仓库(二创版默认),不做更新检查,避免误报上游版本
	if githubRepo == "" {
		cache.Default.PutDuration("github#releases-latest", about, 60*time.Second)
		return about
	}
	req, err := http.NewRequest("GET", "https://api.github.com/repos/"+githubRepo+"/releases/latest", nil)
	if err != nil {
		return about
	}
	req.Header.Set("User-Agent", util.UserAgent())
	if cfg.GithubToken != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.GithubToken)
	}
	client := util.ClientFor(3)
	resp, err := client.Do(req)
	if err != nil {
		return about
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return about
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return about
	}
	b, _ := io.ReadAll(resp.Body)
	var release GithubRelease
	if err := json.Unmarshal(b, &release); err != nil {
		return about
	}
	if release.Message != "" {
		return about
	}
	latest := strings.TrimPrefix(release.TagName, "v")
	about.Latest = latest
	about.MarkdownBody = release.Body
	if t, err := time.Parse(time.RFC3339, release.PublishedAt); err == nil {
		about.Date = model.DateTime(t)
	}
	if versionGreater(latest, about.Version) {
		about.Update = true
	}
	// find the download asset for this platform (jar / exe)
	target := "ani-rss.jar"
	for _, a := range release.Assets {
		if a.Name == target {
			about.DownloadURL = a.BrowserDownloadURL
			about.SHA256 = strings.TrimPrefix(a.Digest, "sha256:")
			about.Size = a.Size
			about.FormatSize = util.FormatSize(a.Size)
			break
		}
	}
	cache.Default.PutDuration("github#releases-latest", about, 60*time.Second)
	return about
}

// versionGreater compares two dotted numeric versions (a > b).
func versionGreater(a, b string) bool {
	pa := parseVersion(a)
	pb := parseVersion(b)
	for i := 0; i < len(pa) || i < len(pb); i++ {
		va, vb := 0, 0
		if i < len(pa) {
			va = pa[i]
		}
		if i < len(pb) {
			vb = pb[i]
		}
		if va != vb {
			return va > vb
		}
	}
	return false
}

func parseVersion(s string) []int {
	s = strings.TrimLeft(s, "vV")
	var out []int
	for _, part := range strings.Split(s, ".") {
		n, _ := strconv.Atoi(part)
		out = append(out, n)
	}
	return out
}

var _ = fmt.Sprintf