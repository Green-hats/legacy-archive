package download

import (
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"ani-rss/internal/config"
	"ani-rss/internal/model"
	"ani-rss/internal/notify"
	"ani-rss/internal/util"
)

// Pan115 is a cloud offline-download client for 115 网盘, authenticated with
// the browser cookie (UID/CID/SEID/KID). Files are downloaded to the 115
// cloud; existence checks and playback go through the web API (CloudClient).
type Pan115 struct {
	mu     sync.Mutex
	client *http.Client
}

const ua115 = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36"

func (p *Pan115) httpClient() *http.Client {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.client == nil {
		p.client = util.ClientFor(20)
	}
	return p.client
}

func (p *Pan115) cookie() string {
	return config.Get().Pan115Cookie
}

// request performs a 115 web API call with the account cookie.
func (p *Pan115) request(method, rawURL string, form url.Values) (map[string]interface{}, error) {
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	req, err := http.NewRequest(method, rawURL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", ua115)
	req.Header.Set("Cookie", p.cookie())
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	resp, err := p.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// Login verifies the 115 cookie by listing the root directory.
func (p *Pan115) Login(test bool, cfg *model.Config) bool {
	if strings.TrimSpace(cfg.Pan115Cookie) == "" {
		warnOnce("pan115-missing-config", "115 未配置完成")
		SetLoginStatus(LoginStatus{Configured: false, Message: "115 未配置 Cookie"})
		return false
	}
	m, err := p.request("GET", "https://webapi.115.com/files?aid=1&cid=0&o=user_ptime&asc=0&offset=0&show_dir=1&limit=1&show_all=0", nil)
	if err != nil {
		utilLogErr("登录 115 失败 %v", err)
		SetLoginStatus(LoginStatus{Configured: true, Message: "115 登录失败: " + err.Error()})
		return false
	}
	state, _ := m["state"].(bool)
	if !state {
		utilLogErr("115 Cookie 无效或已过期")
		SetLoginStatus(LoginStatus{Configured: true, Message: "115 Cookie 无效或已过期"})
		return false
	}
	SetLoginStatus(LoginStatus{Configured: true, OK: true})
	return true
}

// GetTorrentsInfos is unsupported by 115 (cloud tasks).
func (p *Pan115) GetTorrentsInfos() []*model.TorrentsInfo { return nil }

// listFiles returns the entries in a directory (cid).
func (p *Pan115) listFiles(cid string) ([]CloudFile, error) {
	rawURL := fmt.Sprintf("https://webapi.115.com/files?aid=1&cid=%s&o=user_ptime&asc=0&offset=0&show_dir=1&limit=1000&show_all=0", cid)
	m, err := p.request("GET", rawURL, nil)
	if err != nil {
		return nil, err
	}
	data, _ := m["data"].([]interface{})
	var out []CloudFile
	for _, item := range data {
		im, _ := item.(map[string]interface{})
		cloudID := strVal(im["cid"])
		fc, _ := im["fc"].(float64)
		isDir := fc == 0
		if !isDir {
			cloudID = strVal(im["fid"])
		}
		out = append(out, CloudFile{
			Name:     strVal(im["n"]),
			Size:     int64Val(im["s"]),
			IsDir:    isDir,
			ID:       cloudID,
			PickCode: strVal(im["pc"]),
		})
	}
	return out, nil
}

// findDirID locates the folder id for a cloud path (empty if missing).
func (p *Pan115) findDirID(path string) string {
	cid := "0"
	for _, seg := range strings.Split(strings.Trim(path, "/"), "/") {
		if seg == "" {
			continue
		}
		files, err := p.listFiles(cid)
		if err != nil {
			return ""
		}
		found := ""
		for _, f := range files {
			if f.IsDir && f.Name == seg {
				found = f.ID
				break
			}
		}
		if found == "" {
			return ""
		}
		cid = found
	}
	return cid
}

// mkdir creates a folder and returns its id.
func (p *Pan115) mkdir(pid, name string) string {
	form := url.Values{}
	form.Set("pid", pid)
	form.Set("cname", name)
	m, err := p.request("POST", "https://webapi.115.com/files/add", form)
	if err != nil {
		return ""
	}
	if id, ok := m["cid"].(string); ok {
		return id
	}
	if data, ok := m["data"].(map[string]interface{}); ok {
		return strVal(data["cid"])
	}
	return ""
}

// ensureDir returns the folder id for a path, creating directories if needed.
func (p *Pan115) ensureDir(path string) string {
	path = strings.Trim(path, "/")
	if path == "" {
		return "0"
	}
	if id := p.findDirID(path); id != "" {
		return id
	}
	// create level by level
	parent := "0"
	prefix := ""
	for _, seg := range strings.Split(path, "/") {
		if seg == "" {
			continue
		}
		if prefix == "" {
			prefix = seg
		} else {
			prefix = prefix + "/" + seg
		}
		id := p.findDirID(prefix)
		if id == "" {
			id = p.mkdir(parent, seg)
		}
		parent = id
	}
	return parent
}

// Download adds a magnet offline download to 115 and waits for completion.
func (p *Pan115) Download(ani *model.Ani, item *model.Item, savePath, torrentPath string) bool {
	magnet := GetMagnet(torrentPath)
	if magnet == "" {
		return false
	}
	reName := item.ReName
	cloudPath := strings.TrimSuffix(savePath, "/") + "/" + reName
	dirID := p.ensureDir(dirOf(cloudPath))
	if dirID == "" {
		utilLogErr("115 创建目录失败 %s", cloudPath)
		return false
	}
	form := url.Values{}
	form.Set("url", magnet)
	form.Set("wp_path_id", dirID)
	m, err := p.request("POST", "https://115.com/web/lixian/?ct=lixian&ac=add_task_url", form)
	if err != nil {
		utilLogErr("115 添加离线下载失败 %s %v", reName, err)
		return false
	}
	state, _ := m["state"].(bool)
	if !state {
		// errcode 10008 = 任务已存在(此前已添加过)→ 视为已添加,继续等待
		errcode := int64Val(m["errcode"])
		if errcode != 10008 {
			utilLogErr("115 添加离线下载失败 %s %v", reName, m["error_msg"])
			return false
		}
		utilLogInfo("115 任务已存在,等待下载 %s", reName)
	}
	utilLogInfo("115 添加离线下载成功 %s", reName)

	timeout := config.Get().DownloadTimeout
	if timeout <= 0 {
		timeout = 60
	}
	deadline := time.Now().Add(time.Duration(timeout) * time.Minute)
	hash := strings.ToLower(strings.TrimSuffix(baseOf(torrentPath), filepath.Ext(torrentPath)))
	for time.Now().Before(deadline) {
		list, _ := p.request("POST", "https://115.com/web/lixian/?ct=lixian&ac=task_lists", url.Values{"page": {"1"}})
		tasks := offlineTasks(list)
		for _, t := range tasks {
			tm, _ := t.(map[string]interface{})
			name := strVal(tm["name"])
			infoHash := strVal(tm["info_hash"])
			if name == reName || infoHashMatches(hash, infoHash) {
				status := int64Val(tm["status"])
				if status == 2 { // success
					p.notifyDone(ani, item)
					return true
				}
				if status == 3 || status == -1 { // failed
					utilLogErr("115 离线下载失败 %s", reName)
					return false
				}
			}
		}
		// fallback: the task may be out of the list already — check the cloud dir
		if p.FileExists(cloudPath) {
			p.notifyDone(ani, item)
			return true
		}
		time.Sleep(5 * time.Second)
	}
	utilLogErr("115 下载超时 %s", reName)
	return false
}

func (p *Pan115) notifyDone(ani *model.Ani, item *model.Item) {
	text := item.ReName + " 下载完成"
	if !item.Master {
		text = "(备用RSS) " + text
	}
	notify.Send(config.Get(), ani, text, model.NotifyDownloadEnd)
}

// Delete is unsupported by 115.
func (p *Pan115) Delete(t *model.TorrentsInfo, deleteFiles bool) bool { return false }

// SetSavePath is unsupported by 115.
func (p *Pan115) SetSavePath(t *model.TorrentsInfo, path string) {}

// lookupFile returns the pickcode for a cloud path (or "").
func (p *Pan115) lookupPickCode(cloudPath string) string {
	dirID := p.findDirID(dirOf(cloudPath))
	if dirID == "" {
		return ""
	}
	files, err := p.listFiles(dirID)
	if err != nil {
		return ""
	}
	name := baseOf(cloudPath)
	for _, f := range files {
		if !f.IsDir && f.Name == name {
			return f.PickCode
		}
	}
	return ""
}

// FileExists checks whether a cloud file exists at the given cloud path.
func (p *Pan115) FileExists(cloudPath string) bool {
	return p.lookupPickCode(cloudPath) != ""
}

// FileURL returns a playable URL for the cloud file.
func (p *Pan115) FileURL(cloudPath string) string {
	pc := p.lookupPickCode(cloudPath)
	if pc == "" {
		return ""
	}
	rawURL := "https://proapi.115.com/app/chrome/down?pickcode=" + url.QueryEscape(pc) + "&method=get_file_url"
	m, err := p.request("GET", rawURL, nil)
	if err != nil {
		return ""
	}
	data, ok := m["data"].(map[string]interface{})
	if !ok {
		return ""
	}
	for _, v := range data {
		fm, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		um, ok := fm["url"].(map[string]interface{})
		if !ok {
			continue
		}
		if u := strVal(um["url"]); u != "" {
			return u
		}
	}
	return ""
}

// DeleteDir removes a cloud folder at the given cloud path (recursively).
func (p *Pan115) DeleteDir(cloudPath string) bool {
	cid := "0"
	parent := ""
	for _, seg := range strings.Split(strings.Trim(cloudPath, "/"), "/") {
		if seg == "" {
			continue
		}
		files, err := p.listFiles(cid)
		if err != nil {
			return false
		}
		found := ""
		for _, f := range files {
			if f.IsDir && f.Name == seg {
				found = f.ID
				break
			}
		}
		if found == "" {
			return false // 目录不存在
		}
		parent = cid
		cid = found
	}
	if cid == "0" || parent == "" {
		return false
	}
	form := url.Values{}
	form.Set("pid", parent)
	form.Set("fid[0]", cid)
	m, err := p.request("POST", "https://webapi.115.com/rb/delete", form)
	if err != nil {
		return false
	}
	state, _ := m["state"].(bool)
	return state
}

// ListDir lists the files in a cloud directory.
func (p *Pan115) ListDir(cloudPath string) []CloudFile {
	dirID := p.findDirID(cloudPath)
	if dirID == "" {
		return nil
	}
	files, err := p.listFiles(dirID)
	if err != nil {
		return nil
	}
	var out []CloudFile
	for _, f := range files {
		out = append(out, f)
	}
	return out
}

// offlineTasks extracts the offline task list from a task_lists response
// (115 returns the tasks at the top level, sometimes wrapped in "data").
func offlineTasks(m map[string]interface{}) []interface{} {
	if tasks, ok := m["tasks"].([]interface{}); ok {
		return tasks
	}
	if data, ok := m["data"].(map[string]interface{}); ok {
		tasks, _ := data["tasks"].([]interface{})
		return tasks
	}
	return nil
}

// infoHashMatches reports whether a local torrent hash (magnet base32 or hex
// form) matches the hex info_hash reported by a 115 offline task.
func infoHashMatches(local, remote string) bool {
	local = strings.ToLower(strings.TrimSpace(local))
	remote = strings.ToLower(strings.TrimSpace(remote))
	if local == "" || remote == "" {
		return false
	}
	if local == remote {
		return true
	}
	for _, s := range []string{local, remote} {
		if h := b32ToHex(s); h != "" && (h == local || h == remote) {
			return true
		}
	}
	return false
}

// b32ToHex decodes a 32-char base32 magnet info hash into its hex form
// ("" when s is not a valid base32 btih).
func b32ToHex(s string) string {
	if len(s) != 32 {
		return ""
	}
	raw, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(s))
	if err != nil {
		return ""
	}
	return hex.EncodeToString(raw)
}

func strVal(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	if f, ok := v.(float64); ok {
		return fmt.Sprintf("%d", int64(f))
	}
	return ""
}

func int64Val(v interface{}) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case string:
		var n int64
		for _, c := range t {
			if c < '0' || c > '9' {
				return n
			}
			n = n*10 + int64(c-'0')
		}
		return n
	}
	return 0
}