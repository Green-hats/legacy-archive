package notify

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"ani-rss/internal/model"
)

var upgrader = websocket.Upgrader{}

// TestQqListener runs the full capture flow: mock gateway issues a
// C2C_MESSAGE_CREATE event, the listener captures the openid, saves it via
// SaveNotificationConfig and sends an auto-reply through the mock message API.
func TestQqListener(t *testing.T) {
	var mu sync.Mutex
	var sentReplies []string
	var capturedOpenid string
	saved := false

	mux := http.NewServeMux()
	mux.HandleFunc("/app/getAppAccessToken", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"mock_token","expires_in":7200}`))
	})
	mux.HandleFunc("/gateway", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"url":"ws://` + r.Host + `/ws"}`))
	})
mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("ws upgrade err: %v", err)
			return
		}
		defer conn.Close()
		// hello
		if err := conn.WriteJSON(map[string]interface{}{"op": 10, "d": map[string]interface{}{"heartbeat_interval": 30000}}); err != nil {
			t.Logf("ws write hello err: %v", err)
			return
		}
		t.Log("ws: hello sent, waiting a moment for identify")
		time.Sleep(300 * time.Millisecond)
		// send C2C event
		event, _ := json.Marshal(map[string]interface{}{
			"op": 0, "s": 1, "t": "C2C_MESSAGE_CREATE",
			"d": map[string]interface{}{
				"id":      "MSG_ID_1",
				"author":  map[string]interface{}{"user_openid": "USER_OPENID_TEST_123"},
				"content": "你好",
			},
		})
		if err := conn.WriteMessage(websocket.TextMessage, event); err != nil {
			t.Logf("ws write event err: %v", err)
			return
		}
		t.Log("ws: event sent, waiting for close")
		// read until close
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	})
	mux.HandleFunc("/v2/", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		sentReplies = append(sentReplies, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"ROBOT1.0_reply","timestamp":"2026-08-22T18:00:00+08:00"}`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	SaveNotificationConfig = func(cfg *model.NotificationConfig) error {
		mu.Lock()
		capturedOpenid = cfg.QqTargetId
		mu.Unlock()
		saved = true
		return nil
	}
	CurrentConfig = func() *model.Config { return &model.Config{} }

	cfg := &model.NotificationConfig{
		Enable:               true,
		NotificationType:     model.NotifyQQ,
		QqBotAppId:           "test_app",
		QqBotAppSecret:       "test_secret",
		QqTargetType:         "c2c",
		QqApiBase:            srv.URL,
		NotificationTemplate: "${notification}",
	}
	l := &qqListener{cfg: cfg, stop: make(chan struct{})}
	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := l.connect(); err != nil {
			t.Errorf("connect: %v", err)
		}
	}()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		ok := saved && len(sentReplies) > 0
		replies := len(sentReplies)
		savedNow := saved
		mu.Unlock()
		if ok {
			break
		}
		if time.Now().After(deadline) {
			t.Logf("超时: saved=%v replies=%d", savedNow, replies)
		}
		time.Sleep(50 * time.Millisecond)
	}
	close(l.stop)
	<-done

	mu.Lock()
	defer mu.Unlock()
	if !saved {
		t.Fatal("openid 未通过 SaveNotificationConfig 保存")
	}
	if capturedOpenid != "USER_OPENID_TEST_123" {
		t.Errorf("捕获的 openid 错误: %q", capturedOpenid)
	}
	if len(sentReplies) == 0 {
		t.Fatal("未收到自动回复请求")
	}
	if sentReplies[0] != "/v2/users/USER_OPENID_TEST_123/messages" {
		t.Errorf("自动回复路径错误: %s", sentReplies[0])
	}
}