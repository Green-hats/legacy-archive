package notify

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"ani-rss/internal/log"
	"ani-rss/internal/model"
	"ani-rss/internal/util"
)

// QQ official bot intents.
const (
	intentGroupAndC2C = 1 << 25 // GROUP_AND_C2C_EVENT: 群聊 + 单聊消息
)

const (
	opDispatch    = 0
	opHeartbeat   = 1
	opIdentify    = 2
	opHello       = 10
	opHeartbeatAck = 11
)

// qqListener receives QQ events over WebSocket. When a user messages the bot,
// the listener captures their user_openid, auto-fills the matching
// notification config's qqTargetId, and replies with a test message.
type qqListener struct {
	cfg *model.NotificationConfig
	ws  *websocket.Conn
	seq int64
	stop chan struct{}
}

var (
	qqListenersMu sync.Mutex
	qqListeners   = map[string]*qqListener{} // key: appId|target
)

// SyncQQListeners (re)starts WebSocket listeners for all enabled QQ
// notification configs. Stops listeners whose config was removed/disabled.
func SyncQQListeners() {
	configs := qqEnabledConfigs()
	active := map[string]bool{}
	for i := range configs {
		key := qqListenerKey(&configs[i])
		active[key] = true
		qqListenersMu.Lock()
		_, ok := qqListeners[key]
		qqListenersMu.Unlock()
		if ok {
			continue
		}
		l := &qqListener{cfg: &configs[i], stop: make(chan struct{})}
		qqListenersMu.Lock()
		qqListeners[key] = l
		qqListenersMu.Unlock()
		go l.run()
	}
	// stop stale listeners
	qqListenersMu.Lock()
	for key, l := range qqListeners {
		if !active[key] {
			close(l.stop)
			delete(qqListeners, key)
		}
	}
	qqListenersMu.Unlock()
}

func qqListenerKey(cfg *model.NotificationConfig) string {
	return cfg.QqBotAppId + "|" + cfg.QqTargetType
}

// qqEnabledConfigs returns a snapshot of enabled QQ notification configs.
func qqEnabledConfigs() []model.NotificationConfig {
	var out []model.NotificationConfig
	cfg := CurrentConfig()
	if cfg == nil {
		return out
	}
	for _, nc := range cfg.NotificationConfigList {
		if nc.NotificationType == model.NotifyQQ && nc.Enable && nc.QqBotAppId != "" && nc.QqBotAppSecret != "" {
			out = append(out, nc)
		}
	}
	return out
}

// gatewayBase mirrors the message API base for sandbox/production.
func (l *qqListener) gatewayBase() string {
	if l.cfg.QqApiBase != "" {
		return l.cfg.QqApiBase
	}
	if l.cfg.QqSandbox {
		return "https://sandbox.api.sgroup.qq.com"
	}
	return "https://api.sgroup.qq.com"
}

// run connects to the QQ gateway and processes events until stopped.
func (l *qqListener) run() {
	for {
		select {
		case <-l.stop:
			return
		default:
		}
		if err := l.connect(); err != nil {
			log.Warnf("qq", "QQ 网关连接失败: %v, 30s 后重试", err)
			select {
			case <-l.stop:
				return
			case <-time.After(30 * time.Second):
			}
		}
	}
}

func (l *qqListener) connect() error {
	token, err := qqAccessToken(l.cfg)
	if err != nil {
		return err
	}
	// resolve gateway url
	req, err := http.NewRequest("GET", l.gatewayBase()+"/gateway", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "QQBot "+token)
	resp, err := util.DefaultClient().Do(req)
	if err != nil {
		return err
	}
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("获取网关失败 http %s: %s", resp.Status, string(b))
	}
	var gw struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(b, &gw); err != nil || gw.URL == "" {
		return fmt.Errorf("解析网关失败: %s", string(b))
	}

	ws, _, err := websocket.DefaultDialer.Dial(gw.URL, nil)
	if err != nil {
		return err
	}
	l.ws = ws
	// close the socket when the listener is stopped, unblocking ReadMessage
	stopWatch := make(chan struct{})
	go func() {
		select {
		case <-l.stop:
			ws.Close()
		case <-stopWatch:
		}
	}()
	defer close(stopWatch)
	defer ws.Close()

	// read hello
	if err := l.readHello(ws); err != nil {
		return err
	}
	// identify
	identify, _ := json.Marshal(map[string]interface{}{
		"op": opIdentify,
		"d": map[string]interface{}{
			"token":   "QQBot " + token,
			"intents": intentGroupAndC2C,
			"shard":   []int{0, 1},
		},
	})
	if err := ws.WriteMessage(websocket.TextMessage, identify); err != nil {
		return err
	}

	heartbeatTicker := time.NewTicker(30 * time.Second)
	defer heartbeatTicker.Stop()

	// wait for ready, then loop
	for {
		select {
		case <-l.stop:
			return nil
		case <-heartbeatTicker.C:
			hb, _ := json.Marshal(map[string]interface{}{"op": opHeartbeat, "d": l.seq})
			_ = ws.WriteMessage(websocket.TextMessage, hb)
		default:
		}
		_, msg, err := ws.ReadMessage()
		if err != nil {
			select {
			case <-l.stop:
				return nil
			default:
				return err
			}
		}
		var frame struct {
			Op int             `json:"op"`
			Seq int64          `json:"s"`
			Type string        `json:"t"`
			Data json.RawMessage `json:"d"`
		}
		if err := json.Unmarshal(msg, &frame); err != nil {
			continue
		}
		switch frame.Op {
		case opHeartbeatAck:
			// ok
		case opDispatch:
			if frame.Seq > 0 {
				l.seq = frame.Seq
			}
			l.handleEvent(frame.Type, frame.Data)
		}
	}
}

func (l *qqListener) readHello(ws *websocket.Conn) error {
	ws.SetReadDeadline(time.Now().Add(15 * time.Second))
	_, msg, err := ws.ReadMessage()
	ws.SetReadDeadline(time.Time{})
	if err != nil {
		return err
	}
	var hello struct {
		Op int `json:"op"`
		D  struct {
			HeartbeatInterval int `json:"heartbeat_interval"`
		} `json:"d"`
	}
	if err := json.Unmarshal(msg, &hello); err != nil {
		return err
	}
	return nil
}

// handleEvent processes incoming dispatch events.
func (l *qqListener) handleEvent(eventType string, data json.RawMessage) {
	switch eventType {
	case "C2C_MESSAGE_CREATE":
		var e struct {
			ID     string `json:"id"`
			Author struct {
				UserOpenid string `json:"user_openid"`
			} `json:"author"`
			Content string `json:"content"`
		}
		if err := json.Unmarshal(data, &e); err != nil {
			return
		}
		if e.Author.UserOpenid == "" {
			return
		}
		l.captureOpenid(e.Author.UserOpenid, e.ID)
	case "GROUP_AT_MESSAGE_CREATE":
		// capture group_openid + member_openid for future group target
		var e struct {
			GroupOpenid string `json:"group_openid"`
		}
		if err := json.Unmarshal(data, &e); err != nil {
			return
		}
		if e.GroupOpenid != "" && l.cfg.QqTargetId == "" {
			l.cfg.QqTargetType = "group"
			l.cfg.QqTargetId = e.GroupOpenid
			_ = SaveNotificationConfig(l.cfg)
			log.Infof("qq", "已自动捕获群 openid: %s", e.GroupOpenid)
		}
	}
}

// captureOpenid auto-fills qqTargetId for c2c and replies a test message.
func (l *qqListener) captureOpenid(openid, msgID string) {
	if l.cfg.QqTargetId != openid {
		l.cfg.QqTargetType = "c2c"
		l.cfg.QqTargetId = openid
		_ = SaveNotificationConfig(l.cfg)
	}
	log.Infof("qq", "已自动捕获单聊 openid: %s", openid)
	reply := "✅ ani-rss 已捕获你的 openid,配置已自动保存。这是来自 ani-rss 的测试通知。"
	if err := l.SendDirect(openid, reply); err != nil {
		log.Errorf("qq", "自动回复失败: %v", err)
	}
}

// SendDirect sends a raw message to a c2c target (used for the auto-reply).
func (l *qqListener) SendDirect(openid, content string) error {
	prev := l.cfg.QqTargetId
	l.cfg.QqTargetId = openid
	defer func() { l.cfg.QqTargetId = prev }()
	q := &QQ{}
	return q.Send(l.cfg, model.DefaultAni(), content, model.NotifyDownloadEnd)
}

// SaveNotificationConfig persists the notification config list (wired by main).
var SaveNotificationConfig = func(cfg *model.NotificationConfig) error { return nil }

var _ = strings.TrimSpace