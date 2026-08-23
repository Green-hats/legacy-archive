package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"ani-rss/internal/model"
	"ani-rss/internal/util"
)

// QQ sends messages through the QQ 开放平台 official bot API (v2).
//
// Prerequisites (https://bot.q.qq.com):
//   - AppID + AppSecret from the 开放平台 console
//   - The target openid must be obtained from a real event (the user must
//     message the bot first), ids cannot be invented.
type QQ struct{}

var (
	qqTokenMu    sync.Mutex
	qqTokenCache struct {
		appId    string
		secret   string
		token    string
		expireAt time.Time
	}
)

func qqBase(cfg *model.NotificationConfig) string {
	if cfg.QqApiBase != "" {
		return cfg.QqApiBase
	}
	if cfg.QqSandbox {
		return "https://sandbox.api.sgroup.qq.com"
	}
	return "https://api.sgroup.qq.com"
}

// qqTokenBase returns the host used to fetch access tokens.
func qqTokenBase(cfg *model.NotificationConfig) string {
	if cfg.QqApiBase != "" {
		return cfg.QqApiBase
	}
	return "https://bots.qq.com"
}

// qqAccessToken fetches and caches the access token (expires_in ~7200s).
func qqAccessToken(cfg *model.NotificationConfig) (string, error) {
	qqTokenMu.Lock()
	defer qqTokenMu.Unlock()
	if qqTokenCache.appId == cfg.QqBotAppId &&
		qqTokenCache.secret == cfg.QqBotAppSecret &&
		qqTokenCache.token != "" &&
		time.Now().Before(qqTokenCache.expireAt) {
		return qqTokenCache.token, nil
	}
	body, _ := json.Marshal(map[string]string{
		"appId":        cfg.QqBotAppId,
		"clientSecret": cfg.QqBotAppSecret,
	})
	req, err := http.NewRequest("POST", qqTokenBase(cfg)+"/app/getAppAccessToken", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := util.DefaultClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("QQ 获取凭证失败 http %s: %s", resp.Status, string(b))
	}
	var m struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return "", err
	}
	if m.AccessToken == "" {
		return "", fmt.Errorf("QQ 获取凭证失败: %s", string(b))
	}
	ttl := m.ExpiresIn
	if ttl <= 0 {
		ttl = 7200
	}
	qqTokenCache.appId = cfg.QqBotAppId
	qqTokenCache.secret = cfg.QqBotAppSecret
	qqTokenCache.token = m.AccessToken
	qqTokenCache.expireAt = time.Now().Add(time.Duration(ttl) * time.Second)
	return m.AccessToken, nil
}

// Send posts a text message to the configured target.
func (q *QQ) Send(cfg *model.NotificationConfig, ani *model.Ani, text string, status model.NotificationStatusEnum) error {
	if cfg.QqBotAppId == "" || cfg.QqBotAppSecret == "" || cfg.QqTargetId == "" {
		return fmt.Errorf("QQ 未配置完成")
	}
	token, err := qqAccessToken(cfg)
	if err != nil {
		return err
	}

	targetType := cfg.QqTargetType
	if targetType == "" {
		targetType = "c2c"
	}
	var endpoint string
	switch targetType {
	case "group":
		endpoint = fmt.Sprintf("/v2/groups/%s/messages", cfg.QqTargetId)
	case "channel":
		endpoint = fmt.Sprintf("/v2/channels/%s/messages", cfg.QqTargetId)
	default: // c2c
		endpoint = fmt.Sprintf("/v2/users/%s/messages", cfg.QqTargetId)
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"msg_type": 0,
		"content":  ReplaceTemplate(ani, cfg, text, status),
		"msg_seq":  rand.Intn(100000) + 1,
	})
	req, err := http.NewRequest("POST", qqBase(cfg)+endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "QQBot "+token)
	req.Header.Set("X-Union-Appid", cfg.QqBotAppId)
	req.Header.Set("Content-Type", "application/json")
	resp, err := util.DefaultClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("QQ 发送失败 http %s: %s", resp.Status, string(b))
	}
	var er struct {
		ErrCode int `json:"err_code"`
	}
	if err := json.Unmarshal(b, &er); err == nil && er.ErrCode != 0 && er.ErrCode != 200 {
		return fmt.Errorf("QQ 发送失败 err_code=%d: %s", er.ErrCode, string(b))
	}
	return nil
}