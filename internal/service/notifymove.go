package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"ani-rss/internal/log"
	"ani-rss/internal/model"
	"ani-rss/internal/util"
)

var seasonEpRe = regexp.MustCompile(`[Ss](\d+)[Ee](\d+(\.5)?)`)

// FileMove implements the FILE_MOVE notification (mirrors FileMoveNotification).
func FileMove(cfg *model.NotificationConfig, ani *model.Ani, _ string) error {
	if cfg == nil || ani == nil {
		return nil
	}
	clone := ani.Clone()
	src := GetDownloadPath(clone)
	ova := clone.Ova
	template := cfg.FileMoveTarget
	if ova {
		template = cfg.FileMoveOvaTarget
	}
	target := GetDownloadPathTemplate(clone, template)
	if target == "" || target == src {
		return nil
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		return err
	}
	if ova {
		return fileMoveOva(src, target, cfg.FileMoveCopyModel)
	}
	if cfg.FileMoveDeleteOldEp {
		deleteOldEpisodeLocal(src, target)
	}
	return fileMoveSeasoned(src, target, cfg.FileMoveCopyModel)
}

func fileMoveOva(src, target string, copyModel bool) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		full := filepath.Join(src, e.Name())
		log.Infof("filemove", "OVA/剧场版 文件移动: %s => %s", full, target)
		if err := copyFile(full, filepath.Join(target, e.Name())); err != nil {
			return err
		}
		if !copyModel {
			_ = os.Remove(full)
		}
	}
	return nil
}

func fileMoveSeasoned(src, target string, copyModel bool) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	targetNames := map[string]int64{}
	if te, err := os.ReadDir(target); err == nil {
		for _, e := range te {
			if e.IsDir() {
				continue
			}
			if fi, err := e.Info(); err == nil {
				targetNames[e.Name()] = fi.Size()
			}
		}
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !util.IsVideo(name) && !util.IsSubtitle(name) {
			continue
		}
		if !seasonEpRe.MatchString(name) {
			continue
		}
		full := filepath.Join(src, name)
		fi, err := e.Info()
		if err != nil {
			continue
		}
		if sz, ok := targetNames[name]; ok && sz == fi.Size() {
			continue
		}
		log.Infof("filemove", "文件移动: %s => %s", full, target)
		if err := copyFile(full, filepath.Join(target, name)); err != nil {
			return err
		}
		if !copyModel {
			_ = os.Remove(full)
		}
	}
	return nil
}

func deleteOldEpisodeLocal(src, target string) {
	srcFiles := map[string]int64{}
	episodeSet := map[string]bool{}
	if se, err := os.ReadDir(src); err == nil {
		for _, e := range se {
			if e.IsDir() {
				continue
			}
			if fi, err := e.Info(); err == nil {
				srcFiles[e.Name()] = fi.Size()
			}
			if (util.IsVideo(e.Name()) || util.IsSubtitle(e.Name())) && seasonEpRe.MatchString(e.Name()) {
				episodeSet[strings.ToUpper(seasonEpRe.FindString(e.Name()))] = true
			}
		}
	}
	entries, err := os.ReadDir(target)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !util.IsVideo(name) && !util.IsSubtitle(name) {
			continue
		}
		if !seasonEpRe.MatchString(name) {
			continue
		}
		if sz, ok := srcFiles[name]; ok {
			if fi, err := e.Info(); err == nil && fi.Size() == sz {
				continue
			}
		}
		episode := strings.ToUpper(seasonEpRe.FindString(name))
		if !episodeSet[episode] {
			continue
		}
		log.Infof("filemove", "因洗版需要删除: %s", filepath.Join(target, name))
		_ = os.Remove(filepath.Join(target, name))
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	_, cerr := io.Copy(out, in)
	out.Close()
	return cerr
}

// ---- OpenListUpload notification ----

type openListUploadClient struct {
	host   string
	apiKey string
}

func (c *openListUploadClient) post(path string, body interface{}) (map[string]interface{}, error) {
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest("POST", c.host+"/api/"+path, rdr)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	time.Sleep(2 * time.Second)
	resp, err := util.DefaultClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	var m map[string]interface{}
	_ = json.Unmarshal(b, &m)
	return m, nil
}

type olFileInfo struct {
	Name string  `json:"name"`
	Size int64   `json:"size"`
	IsDir bool   `json:"is_dir"`
}

func (c *openListUploadClient) list(dir string) []olFileInfo {
	body := map[string]interface{}{"path": dir, "password": "", "page": 1, "per_page": 0, "refresh": false}
	m, err := c.post("fs/list", body)
	if err != nil {
		return nil
	}
	data, ok := m["data"].(map[string]interface{})
	if !ok {
		return nil
	}
	content, _ := data["content"].([]interface{})
	var out []olFileInfo
	for _, item := range content {
		im, _ := item.(map[string]interface{})
		out = append(out, olFileInfo{
			Name:   strVal(im["name"]),
			Size:   int64Val(im["size"]),
			IsDir:  boolVal(im["is_dir"]),
		})
	}
	return out
}

func (c *openListUploadClient) remove(dir string, names []string) {
	body := map[string]interface{}{"dir": dir, "names": names}
	_, _ = c.post("fs/remove", body)
}

func (c *openListUploadClient) put(localPath, cloudPath string) error {
	b, err := os.ReadFile(localPath)
	if err != nil {
		return err
	}
	filename := filepath.Base(localPath)
	filePath := url.PathEscape(cloudPath + "/" + filename)
	req, err := http.NewRequest("PUT", c.host+"/api/fs/put", bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.apiKey)
	req.Header.Set("As-Task", "false")
	req.Header.Set("File-Path", filePath)
	req.Header.Set("Content-Type", "application/octet-stream")
	client := util.ClientFor(120)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("上传失败 状态码:%d", resp.StatusCode)
	}
	rb, _ := io.ReadAll(resp.Body)
	var m map[string]interface{}
	_ = json.Unmarshal(rb, &m)
	if code := int64Val(m["code"]); code != 200 {
		return fmt.Errorf("上传失败 code:%d", code)
	}
	log.Infof("openlist-upload", "OpenList 上传完成 %s", filename)
	return nil
}

// OpenListUpload implements the OPEN_LIST_UPLOAD notification.
func OpenListUpload(cfg *model.NotificationConfig, ani *model.Ani, _ string) error {
	if cfg == nil || ani == nil {
		return nil
	}
	clone := ani.Clone()
	localPath := GetDownloadPath(clone)
	ova := clone.Ova
	template := cfg.OpenListUploadPath
	if ova {
		template = cfg.OpenListUploadOvaPath
	}
	if clone.CustomUploadEnable {
		template = clone.CustomUploadPathTarget
	}
	target := GetDownloadPathTemplate(clone, template)
	target = regexp.MustCompile(`^[A-z]:`).ReplaceAllString(target, "")

	client := &openListUploadClient{host: strings.TrimSuffix(cfg.OpenListUploadHost, "/"), apiKey: cfg.OpenListUploadApiKey}
	if cfg.OpenListUploadHost == "" || cfg.OpenListUploadApiKey == "" {
		return fmt.Errorf("OpenList 上传配置不完整")
	}
	if ova {
		return olUploadOva(client, localPath, target)
	}
	if cfg.OpenListUploadDelOldEp {
		olDeleteOldEpisode(client, localPath, target)
	}
	return olUpload(client, localPath, target, cfg.OpenListUploadDelLocal)
}

func olUploadOva(c *openListUploadClient, local, target string) error {
	entries, err := os.ReadDir(local)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || (!util.IsSubtitle(e.Name()) && !util.IsVideo(e.Name())) {
			continue
		}
		if err := c.put(filepath.Join(local, e.Name()), target); err != nil {
			return err
		}
	}
	return nil
}

func olDeleteOldEpisode(c *openListUploadClient, local, target string) {
	localFiles := map[string]int64{}
	episodeSet := map[string]bool{}
	if se, err := os.ReadDir(local); err == nil {
		for _, e := range se {
			if e.IsDir() {
				continue
			}
			if fi, err := e.Info(); err == nil {
				localFiles[e.Name()] = fi.Size()
			}
			if (util.IsVideo(e.Name()) || util.IsSubtitle(e.Name())) && seasonEpRe.MatchString(e.Name()) {
				episodeSet[strings.ToUpper(seasonEpRe.FindString(e.Name()))] = true
			}
		}
	}
	for _, fi := range c.list(target) {
		name := fi.Name
		if !util.IsVideo(name) && !util.IsSubtitle(name) {
			continue
		}
		if !seasonEpRe.MatchString(name) {
			continue
		}
		if sz, ok := localFiles[name]; ok && sz == fi.Size {
			continue
		}
		episode := strings.ToUpper(seasonEpRe.FindString(name))
		if !episodeSet[episode] {
			continue
		}
		log.Infof("openlist-upload", "因洗版需要删除: %s", name)
		c.remove(target, []string{name})
	}
}

func olUpload(c *openListUploadClient, local, target string, deleteLocal bool) error {
	cloudFiles := map[string]int64{}
	for _, fi := range c.list(target) {
		cloudFiles[fi.Name] = fi.Size
	}
	entries, err := os.ReadDir(local)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || (!util.IsVideo(e.Name()) && !util.IsSubtitle(e.Name())) {
			continue
		}
		name := e.Name()
		if !seasonEpRe.MatchString(name) {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			continue
		}
		if sz, ok := cloudFiles[name]; ok && sz == fi.Size() {
			continue
		}
		log.Infof("openlist-upload", "文件上传: %s => %s", filepath.Join(local, name), target)
		if err := c.put(filepath.Join(local, name), target); err != nil {
			return err
		}
		if deleteLocal {
			_ = os.Remove(filepath.Join(local, name))
		}
	}
	return nil
}

func strVal(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func int64Val(v interface{}) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case int:
		return int64(t)
	}
	return 0
}

func boolVal(v interface{}) bool {
	b, _ := v.(bool)
	return b
}