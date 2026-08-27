package server

import (
	"net/http"

	"ani-rss/internal/bgm"
	"ani-rss/internal/config"
	"ani-rss/internal/model"
	"ani-rss/internal/service"
)

// handleSearchBgm processes POST /api/searchBgm.
func (s *Server) handleSearchBgm(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		fail(w, "name 不能为空")
		return
	}
	ok(w, bgm.Search(name))
}

// handleGetBgmTitle processes POST /api/getBgmTitle.
func (s *Server) handleGetBgmTitle(w http.ResponseWriter, r *http.Request) {
	var body model.Ani
	if !readJSONOrFail(w, r, &body) {
		return
	}
	subjectId := service.GetSubjectIdFromAni(&body)
	if subjectId == "" {
		fail(w, "未获取到BGM信息")
		return
	}
	info, err := bgm.GetBgmInfo(subjectId)
	if err != nil {
		fail(w, err.Error())
		return
	}
	ok(w, info.Name)
}

// handleRate processes POST /api/rate.
func (s *Server) handleRate(w http.ResponseWriter, r *http.Request) {
	var body model.Ani
	if !readJSONOrFail(w, r, &body) {
		return
	}
	subjectId := service.GetSubjectIdFromAni(&body)
	if subjectId == "" {
		fail(w, "未获取到BGM信息")
		return
	}
	score, err := bgm.GetRate(subjectId)
	if err != nil {
		fail(w, err.Error())
		return
	}
	ok(w, score)
}

// handleSetRate processes POST /api/setRate.
func (s *Server) handleSetRate(w http.ResponseWriter, r *http.Request) {
	var body model.Ani
	if !readJSONOrFail(w, r, &body) {
		return
	}
	subjectId := service.GetSubjectIdFromAni(&body)
	if subjectId == "" {
		fail(w, "未获取到BGM信息")
		return
	}
	if err := bgm.SetRate(subjectId, body.Score); err != nil {
		fail(w, err.Error())
		return
	}
	// update stored ani score
	for _, ani := range config.AniList() {
		if ani != nil && ani.ID == body.ID {
			ani.Score = body.Score
		}
	}
	_ = config.SyncAni()
	okMsg(w, "保存评分成功")
}

// handleMeBgm processes POST /api/meBgm.
func (s *Server) handleMeBgm(w http.ResponseWriter, r *http.Request) {
	me, err := bgm.Me()
	if err != nil {
		fail(w, err.Error())
		return
	}
	ok(w, me)
}

// handleBgmOauthCallback processes POST /api/bgm/oauth/callback.
func (s *Server) handleBgmOauthCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		fail(w, "code 不能为空")
		return
	}
	if err := bgm.ExchangeCode(code); err != nil {
		fail(w, err.Error())
		return
	}
	okMsg(w, "授权成功, 现在你可以关闭此窗口")
}