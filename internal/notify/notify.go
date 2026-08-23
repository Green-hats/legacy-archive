package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"ani-rss/internal/model"
	"ani-rss/internal/util"
)

// Sender is one notification channel implementation.
type Sender interface {
	Send(cfg *model.NotificationConfig, ani *model.Ani, text string, status model.NotificationStatusEnum) error
}

// Send dispatches a notification to all enabled matching channels.
func Send(cfg *model.Config, ani *model.Ani, text string, status model.NotificationStatusEnum) {
	if ani != nil && !ani.Message {
		return
	}
	list := append([]model.NotificationConfig(nil), cfg.NotificationConfigList...)
	sort.SliceStable(list, func(i, j int) bool { return list[i].Sort < list[j].Sort })
	for _, nc := range list {
		if !nc.Enable {
			continue
		}
		matched := false
		for _, s := range nc.StatusList {
			if s == status {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		sender := senderFor(nc.NotificationType)
		if sender == nil {
			continue
		}
		nc := nc
		go func() {
			retry := nc.Retry
			if retry <= 0 {
				retry = 1
			}
			for i := 0; i < retry; i++ {
				if i > 0 {
					time.Sleep(time.Second)
				}
				if err := sender.Send(&nc, ani, text, status); err == nil {
					return
				}
			}
		}()
	}
}

func senderFor(t model.NotificationTypeEnum) Sender {
	return SenderFor(t)
}

// SenderFor returns the sender implementation for a notification type.
func SenderFor(t model.NotificationTypeEnum) Sender {
	switch t {
	case model.NotifyTelegram:
		return &Telegram{}
	case model.NotifyBark:
		return &Bark{}
	case model.NotifyServerChan:
		return &ServerChan{}
	case model.NotifyWebHook:
		return &WebHook{}
	case model.NotifyShell:
		return &Shell{}
	case model.NotifySystem:
		return &SystemNotifier{}
	case model.NotifyMail:
		return &Mail{}
	case model.NotifyEmbyRefresh:
		return &EmbyRefresh{}
	case model.NotifyFileMove:
		return &FileMove{}
	case model.NotifyOpenListUpload:
		return &OpenListUpload{}
	}
	return nil
}

// ReplaceTemplate fills the notification template placeholders.
func ReplaceTemplate(ani *model.Ani, nc *model.NotificationConfig, text string, status model.NotificationStatusEnum) string {
	cfg := CurrentConfig()
	tmpl := nc.NotificationTemplate
	if tmpl == "" {
		tmpl = "${notification}"
	}
	tmpl = replaceBase(tmpl, ani, text, status, nc.Comment, cfg.DownloadPathTemplate)
	if strings.Contains(tmpl, "${notification}") {
		t := cfg.NotificationTemplate
		if t == "" {
			t = "${text}"
		}
		t = replaceBase(t, ani, text, status, nc.Comment, cfg.DownloadPathTemplate)
		tmpl = strings.ReplaceAll(tmpl, "${notification}", t)
	}
	return strings.TrimSpace(tmpl)
}

func replaceBase(tmpl string, ani *model.Ani, text string, status model.NotificationStatusEnum, comment, downloadPath string) string {
	tmpl = strings.ReplaceAll(tmpl, "${text}", text)
	tmpl = strings.ReplaceAll(tmpl, "${comment}", comment)
	if ani != nil {
		tmpl = strings.ReplaceAll(tmpl, "${title}", ani.Title)
		tmpl = strings.ReplaceAll(tmpl, "${season}", strconv.Itoa(ani.Season))
		tmpl = strings.ReplaceAll(tmpl, "${seasonFormat}", fmt.Sprintf("%02d", ani.Season))
		tmpl = strings.ReplaceAll(tmpl, "${episode}", strconv.Itoa(ani.CurrentEpisodeNumber))
		tmpl = strings.ReplaceAll(tmpl, "${episodeFormat}", fmt.Sprintf("%02d", ani.CurrentEpisodeNumber))
		tmpl = strings.ReplaceAll(tmpl, "${themoviedbName}", ani.ThemoviedbName)
		if ani.Tmdb != nil {
			tmpl = strings.ReplaceAll(tmpl, "${tmdbid}", strconv.Itoa(ani.Tmdb.ID))
		}
		tmpl = strings.ReplaceAll(tmpl, "${jpTitle}", ani.JpTitle)
		tmpl = strings.ReplaceAll(tmpl, "${bgmUrl}", ani.BgmUrl)
		tmpl = strings.ReplaceAll(tmpl, "${url}", ani.URL)
	}
	tmpl = strings.ReplaceAll(tmpl, "${downloadPath}", downloadPath)
	emoji, action := statusMeta(status)
	tmpl = strings.ReplaceAll(tmpl, "${emoji}", emoji)
	tmpl = strings.ReplaceAll(tmpl, "${action}", action)
	return tmpl
}

func statusMeta(status model.NotificationStatusEnum) (string, string) {
	switch status {
	case model.NotifyDownloadStart:
		return "⬇️", "开始下载"
	case model.NotifyDownloadEnd:
		return "✅", "下载完成"
	case model.NotifyOmit:
		return "⚠️", "缺少集数"
	case model.NotifyError:
		return "❌", "错误"
	case model.NotifyCompleted:
		return "🏁", "订阅完成"
	case model.NotifyProcrastinating:
		return "🐟", "摸鱼"
	}
	return "", ""
}

// CurrentConfig returns the active config (set by the service layer).
var CurrentConfig func() *model.Config

// SendRaw posts an HTTP request with optional headers (webhook/telegram/bark).
func sendHTTP(method, rawURL string, headers map[string]string, form url.Values, jsonBody interface{}) error {
	var rdr io.Reader
	contentType := ""
	if form != nil {
		rdr = strings.NewReader(form.Encode())
		contentType = "application/x-www-form-urlencoded"
	} else if jsonBody != nil {
		b, err := json.Marshal(jsonBody)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(b)
		contentType = "application/json"
	}
	req, err := http.NewRequest(method, rawURL, rdr)
	if err != nil {
		return err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := util.DefaultClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("http %s: %s", resp.Status, string(b))
	}
	return nil
}