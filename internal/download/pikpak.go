package download

import (
	"strings"
	"sync"
	"time"

	"github.com/akshardp/pikpak-go"

	"ani-rss/internal/config"
	"ani-rss/internal/model"
	"ani-rss/internal/notify"
)

// PikPak is a cloud offline-download client (PikPak 网盘).
// Files are downloaded to the PikPak cloud; existence checks and playback go
// through the cloud API (CloudClient).
type PikPak struct {
	mu     sync.Mutex
	client *pikpak.Client
}

func (p *PikPak) getClient() (*pikpak.Client, error) {
	cfg := config.Get()
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.client != nil && p.client.IsAuthenticated() {
		return p.client, nil
	}
	if strings.TrimSpace(cfg.PikpakEmail) == "" || strings.TrimSpace(cfg.PikpakPassword) == "" {
		return nil, errNotConfigured
	}
	c, err := pikpak.NewClient(
		pikpak.WithCredentials(cfg.PikpakEmail, cfg.PikpakPassword),
		pikpak.WithPlatform(pikpak.PlatformWeb),
	)
	if err != nil {
		return nil, err
	}
	if err := c.Init(); err != nil {
		return nil, err
	}
	p.client = c
	return c, nil
}

var errNotConfigured = newError("PikPak 未配置完成")

// Login authenticates with the PikPak account.
func (p *PikPak) Login(test bool, cfg *model.Config) bool {
	_, err := p.getClient()
	if err != nil {
		if strings.Contains(err.Error(), "未配置") {
			warnOnce("pikpak-missing-config", "PikPak 未配置完成")
			SetLoginStatus(LoginStatus{Configured: false, Message: "PikPak 未配置"})
		} else {
			utilLogErr("登录 PikPak 失败 %v", err)
			SetLoginStatus(LoginStatus{Configured: true, Message: "PikPak 登录失败: " + err.Error()})
		}
		return false
	}
	SetLoginStatus(LoginStatus{Configured: true, OK: true})
	return true
}

// GetTorrentsInfos is unsupported by PikPak (cloud tasks, no torrent list).
func (p *PikPak) GetTorrentsInfos() []*model.TorrentsInfo { return nil }

// Download adds a magnet/ed2k offline download to PikPak and waits for completion.
func (p *PikPak) Download(ani *model.Ani, item *model.Item, savePath, torrentPath string) bool {
	client, err := p.getClient()
	if err != nil {
		return false
	}
	magnet := GetMagnet(torrentPath)
	if magnet == "" {
		return false
	}
	reName := item.ReName
	cloudPath := strings.TrimSuffix(savePath, "/") + "/" + reName

	folderID, err := client.CreateFolderPath(cloudPath, "")
	if err != nil {
		utilLogErr("PikPak 创建目录失败 %s %v", cloudPath, err)
		return false
	}
	task, err := client.OfflineDownload(magnet, folderID, reName)
	if err != nil {
		utilLogErr("PikPak 添加离线下载失败 %s %v", reName, err)
		return false
	}
	utilLogInfo("PikPak 添加离线下载成功 %s", reName)

	// wait for completion (mirrors OpenList behavior with a generous timeout)
	timeout := config.Get().DownloadTimeout
	if timeout <= 0 {
		timeout = 60
	}
	deadline := time.Now().Add(time.Duration(timeout) * time.Minute)
	for time.Now().Before(deadline) {
		t, err := client.GetOfflineTask(task.ID)
		if err == nil && t != nil {
			switch t.Phase {
			case "PHASE_TYPE_COMPLETE":
				text := item.ReName + " 下载完成"
				if !item.Master {
					text = "(备用RSS) " + text
				}
				notify.Send(config.Get(), ani, text, model.NotifyDownloadEnd)
				return true
			case "PHASE_TYPE_ERROR", "PHASE_TYPE_FAILED":
				utilLogErr("PikPak 离线下载失败 %s", reName)
				return false
			}
		}
		time.Sleep(5 * time.Second)
	}
	utilLogErr("PikPak 下载超时 %s", reName)
	return false
}

// Delete is unsupported by PikPak.
func (p *PikPak) Delete(t *model.TorrentsInfo, deleteFiles bool) bool { return false }

// SetSavePath is unsupported by PikPak.
func (p *PikPak) SetSavePath(t *model.TorrentsInfo, path string) {}

// FileExists checks whether a cloud file exists at the given cloud path.
func (p *PikPak) FileExists(cloudPath string) bool {
	client, err := p.getClient()
	if err != nil {
		return false
	}
	folder, err := client.GetFolderByPath(dirOf(cloudPath))
	if err != nil || folder == nil {
		return false
	}
	files, err := client.GetFolderContents(folder.ID)
	if err != nil {
		return false
	}
	name := baseOf(cloudPath)
	for _, f := range files {
		if f.Name == name {
			return true
		}
	}
	return false
}

// FileURL returns a playable URL for the cloud file.
func (p *PikPak) FileURL(cloudPath string) string {
	client, err := p.getClient()
	if err != nil {
		return ""
	}
	folder, err := client.GetFolderByPath(dirOf(cloudPath))
	if err != nil || folder == nil {
		return ""
	}
	files, err := client.GetFolderContents(folder.ID)
	if err != nil {
		return ""
	}
	name := baseOf(cloudPath)
	for _, f := range files {
		if f.Name == name {
			if url, err := client.GetMediaURL(f.ID); err == nil {
				return url
			}
		}
	}
	return ""
}

// ListDir lists the files in a cloud directory.
func (p *PikPak) ListDir(cloudPath string) []CloudFile {
	client, err := p.getClient()
	if err != nil {
		return nil
	}
	folder, err := client.GetFolderByPath(cloudPath)
	if err != nil || folder == nil {
		return nil
	}
	files, err := client.GetFolderContents(folder.ID)
	if err != nil {
		return nil
	}
	var out []CloudFile
	for _, f := range files {
		out = append(out, CloudFile{Name: f.Name, Size: parseSize(f.Size), IsDir: f.Kind == "folder"})
	}
	return out
}

// DeleteDir is unsupported by PikPak (dev).
func (p *PikPak) DeleteDir(cloudPath string) bool { return false }



func parseSize(s string) int64 {
	var n int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return n
		}
		n = n*10 + int64(c-'0')
	}
	return n
}

func newError(s string) error { return &plainError{s} }

type plainError struct{ s string }

func (e *plainError) Error() string { return e.s }