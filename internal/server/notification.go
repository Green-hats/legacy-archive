package server

import (
	"net/http"

	"ani-rss/internal/config"
	"ani-rss/internal/model"
	"ani-rss/internal/notify"
)

// handleTestNotification processes POST /api/testNotification.
func (s *Server) handleTestNotification(w http.ResponseWriter, r *http.Request) {
	var body model.NotificationConfig
	if !readJSONOrFail(w, r, &body) {
		return
	}
	cfg := config.Get()
	cfg.NotificationConfigList = []model.NotificationConfig{body}
	sender := notify.SenderFor(body.NotificationType)
	if sender == nil {
		fail(w, "未知的通知类型")
		return
	}
	if err := sender.Send(&body, model.DefaultAni(), "测试通知", model.NotifyDownloadEnd); err != nil {
		fail(w, err.Error())
		return
	}
	ok(w, nil)
}

// handleNewNotification processes POST /api/newNotification.
func (s *Server) handleNewNotification(w http.ResponseWriter, r *http.Request) {
	ok(w, model.DefaultNotificationConfig())
}

// handleGetTgUpdates processes POST /api/getTgUpdates.
func (s *Server) handleGetTgUpdates(w http.ResponseWriter, r *http.Request) {
	var body model.NotificationConfig
	if !readJSONOrFail(w, r, &body) {
		return
	}
	updates, err := notify.GetUpdates(&body)
	if err != nil {
		fail(w, err.Error())
		return
	}
	ok(w, updates)
}