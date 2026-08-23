package notify

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"strconv"
	"strings"

	"ani-rss/internal/model"
	"ani-rss/internal/util"
)

// Telegram sends messages via the Bot API.
type Telegram struct{}

func tgHost(cfg *model.NotificationConfig) string {
	if cfg.TelegramApiHost != "" {
		return strings.TrimSuffix(cfg.TelegramApiHost, "/")
	}
	return "https://api.telegram.org"
}

// Send posts a sendMessage (or sendPhoto) request.
func (t *Telegram) Send(cfg *model.NotificationConfig, ani *model.Ani, text string, status model.NotificationStatusEnum) error {
	if cfg.TelegramBotToken == "" || cfg.TelegramChatId == "" {
		return fmt.Errorf("telegram 未配置完成")
	}
	tmpl := ReplaceTemplate(ani, cfg, text, status)
	form := url.Values{}
	form.Set("chat_id", cfg.TelegramChatId)
	if cfg.TelegramTopicId > 0 {
		form.Set("message_thread_id", strconv.Itoa(cfg.TelegramTopicId))
	}
	if cfg.TelegramFormat == "markdown" {
		form.Set("parse_mode", "MarkdownV2")
	} else if cfg.TelegramFormat == "html" {
		form.Set("parse_mode", "HTML")
	}
	if cfg.TelegramImage {
		url2 := fmt.Sprintf("%s/bot%s/sendPhoto", tgHost(cfg), cfg.TelegramBotToken)
		form.Set("caption", tmpl)
		if ani != nil && ani.Image != "" {
			form.Set("photo", ani.Image)
		}
		return sendHTTP("POST", url2, nil, form, nil)
	}
	url2 := fmt.Sprintf("%s/bot%s/sendMessage", tgHost(cfg), cfg.TelegramBotToken)
	form.Set("text", tmpl)
	return sendHTTP("POST", url2, nil, form, nil)
}

// GetUpdates returns the recent chat list for configuration.
func GetUpdates(cfg *model.NotificationConfig) ([]map[string]interface{}, error) {
	if cfg.TelegramBotToken == "" {
		return nil, fmt.Errorf("telegram 未配置完成")
	}
	url2 := fmt.Sprintf("%s/bot%s/getUpdates", tgHost(cfg), cfg.TelegramBotToken)
	resp, err := http.Get(url2)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var body struct {
		OK     bool                     `json:"ok"`
		Result []map[string]interface{} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	return body.Result, nil
}

// Bark sends a push notification.
type Bark struct{}

func (b *Bark) Send(cfg *model.NotificationConfig, ani *model.Ani, text string, status model.NotificationStatusEnum) error {
	server := cfg.BarkServerUrl
	if server == "" {
		server = "https://api.day.app"
	}
	title := statusMetaLabel(status)
	if ani != nil {
		title = ani.Title
	}
	for _, key := range splitKeys(cfg.BarkDeviceKeys) {
		u := fmt.Sprintf("%s/%s", strings.TrimSuffix(server, "/"), key)
		form := url.Values{}
		form.Set("title", title)
		form.Set("body", ReplaceTemplate(ani, cfg, text, status))
		if cfg.BarkGroup != "" {
			form.Set("group", cfg.BarkGroup)
		}
		if cfg.BarkLevel != "" {
			form.Set("level", cfg.BarkLevel)
		}
		if cfg.BarkVolume > 0 {
			form.Set("volume", strconv.Itoa(cfg.BarkVolume))
		}
		if cfg.BarkUseMarkdown {
			form.Set("device_token", key)
		}
		if err := sendHTTP("POST", u, nil, form, nil); err != nil {
			return err
		}
	}
	return nil
}

func splitKeys(s string) []string {
	var out []string
	for _, k := range strings.Split(s, ",") {
		if k = strings.TrimSpace(k); k != "" {
			out = append(out, k)
		}
	}
	return out
}

func statusMetaLabel(status model.NotificationStatusEnum) string {
	_, action := statusMeta(status)
	return action
}

// ServerChan sends via ServerChan.
type ServerChan struct{}

func (s *ServerChan) Send(cfg *model.NotificationConfig, ani *model.Ani, text string, status model.NotificationStatusEnum) error {
	if cfg.ServerChanSendKey == "" {
		return fmt.Errorf("serverChan 未配置完成")
	}
	title := statusMetaLabel(status)
	if ani != nil {
		title = ani.Title
	}
	var u string
	if cfg.ServerChanType == model.ServerChanType3 {
		u = cfg.ServerChan3ApiUrl
		if u == "" {
			u = "https://sctapi.ftqq.com"
		}
		u = strings.TrimSuffix(u, "/") + "/" + cfg.ServerChanSendKey + ".send"
	} else {
		u = "https://sctapi.ftqq.com/" + cfg.ServerChanSendKey + ".send"
	}
	form := url.Values{}
	form.Set("title", title)
	form.Set("desp", ReplaceTemplate(ani, cfg, text, status))
	return sendHTTP("POST", u, nil, form, nil)
}

// WebHook posts to a generic webhook URL.
type WebHook struct{}

func (w *WebHook) Send(cfg *model.NotificationConfig, ani *model.Ani, text string, status model.NotificationStatusEnum) error {
	if cfg.WebHookUrl == "" {
		return fmt.Errorf("webhook 未配置完成")
	}
	method := strings.ToUpper(cfg.WebHookMethod)
	if method == "" {
		method = "POST"
	}
	body := ReplaceTemplate(ani, cfg, text, status)
	var headers map[string]string
	if cfg.WebHookHeader != "" {
		headers = parseHeaders(cfg.WebHookHeader)
	}
	var payload interface{} = body
	if strings.HasPrefix(strings.TrimSpace(body), "{") {
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(body), &m); err == nil {
			payload = m
		}
	}
	return sendHTTP(method, cfg.WebHookUrl, headers, nil, payload)
}

func parseHeaders(s string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(s, "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			out[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return out
}

// Shell runs a shell command.
type Shell struct{}

func (s *Shell) Send(cfg *model.NotificationConfig, ani *model.Ani, text string, status model.NotificationStatusEnum) error {
	if cfg.Shell == "" {
		return fmt.Errorf("shell 未配置完成")
	}
	cmd := exec.Command("bash", "-c", cfg.Shell)
	cmd.Env = append(cmd.Environ(),
		"ANI_RSS_TEXT="+text,
		"ANI_RSS_TITLE="+aniTitle(ani),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("shell: %v %s", err, string(out))
	}
	return nil
}

func aniTitle(ani *model.Ani) string {
	if ani == nil {
		return ""
	}
	return ani.Title
}

// SystemNotifier logs to the internal log (visible in /api/logs).
type SystemNotifier struct{}

func (s *SystemNotifier) Send(cfg *model.NotificationConfig, ani *model.Ani, text string, status model.NotificationStatusEnum) error {
	LogMsg(ReplaceTemplate(ani, cfg, text, status))
	return nil
}

// LogMsg is wired to the internal log package by main.
var LogMsg func(string)

// EmbyRefresh triggers an emby library refresh.
type EmbyRefresh struct{}

func (e *EmbyRefresh) Send(cfg *model.NotificationConfig, ani *model.Ani, text string, status model.NotificationStatusEnum) error {
	if cfg.EmbyHost == "" || cfg.EmbyApiKey == "" {
		return fmt.Errorf("emby 未配置完成")
	}
	for _, viewID := range strings.Split(cfg.EmbyRefreshViewIds, ",") {
		viewID = strings.TrimSpace(viewID)
		if viewID == "" {
			continue
		}
		u := strings.TrimSuffix(cfg.EmbyHost, "/") + "/emby/Library/Refresh?api_key=" + cfg.EmbyApiKey
		_ = viewID
		if err := sendHTTP("POST", u, nil, nil, nil); err != nil {
			return err
		}
	}
	return nil
}

// FileMove moves downloaded files (stub wiring to service layer).
type FileMove struct{}

func (f *FileMove) Send(cfg *model.NotificationConfig, ani *model.Ani, text string, status model.NotificationStatusEnum) error {
	return FileMoveFn(cfg, ani, text)
}

// FileMoveFn is wired to the service layer.
var FileMoveFn func(cfg *model.NotificationConfig, ani *model.Ani, text string) error

// OpenListUpload uploads files to OpenList (stub wiring to service layer).
type OpenListUpload struct{}

func (o *OpenListUpload) Send(cfg *model.NotificationConfig, ani *model.Ani, text string, status model.NotificationStatusEnum) error {
	return OpenListUploadFn(cfg, ani, text)
}

// OpenListUploadFn is wired to the service layer.
var OpenListUploadFn func(cfg *model.NotificationConfig, ani *model.Ani, text string) error

// Mail sends email via SMTP.
type Mail struct{}

func (m *Mail) Send(cfg *model.NotificationConfig, ani *model.Ani, text string, status model.NotificationStatusEnum) error {
	return SendMail(cfg, ani, text, status)
}

var _ = io.Discard
var _ = util.UserAgent