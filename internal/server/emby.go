package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"ani-rss/internal/model"
	"ani-rss/internal/util"
)

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// fetchEmbyViews lists emby media libraries.
func fetchEmbyViews(host, apiKey string) []*model.EmbyViews {
	u := strings.TrimSuffix(host, "/") + "/emby/Users?api_key=" + apiKey
	b, err := util.GetBytes(u)
	if err != nil {
		return nil
	}
	var users []struct {
		ID string `json:"Id"`
	}
	if err := json.Unmarshal(b, &users); err != nil || len(users) == 0 {
		return nil
	}
	uid := users[0].ID
	u2 := strings.TrimSuffix(host, "/") + "/emby/Users/" + uid + "/Views?api_key=" + apiKey
	b, err = util.GetBytes(u2)
	if err != nil {
		return nil
	}
	var body struct {
		Items []struct {
			ID   string `json:"Id"`
			Name string `json:"Name"`
		} `json:"Items"`
	}
	if err := json.Unmarshal(b, &body); err != nil {
		return nil
	}
	var out []*model.EmbyViews
	for _, it := range body.Items {
		out = append(out, &model.EmbyViews{ID: it.ID, Name: it.Name})
	}
	return out
}

// handleEmbyEvent processes an emby webhook event (BGM auto mark).
func handleEmbyEvent(e *model.EmbyWebHook) {
	switch e.Event {
	case "system.webhooktest", "system.notificationtest":
		return
	}
}

var _ = http.MethodGet