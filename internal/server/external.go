package server

import (
	"net/http"
	"strconv"

	"ani-rss/internal/config"
	"ani-rss/internal/mikan"
	"ani-rss/internal/model"
	"ani-rss/internal/scrape"
	"ani-rss/internal/service"
	"ani-rss/internal/tmdb"
)

// handleMikan processes POST /api/mikan.
func (s *Server) handleMikan(w http.ResponseWriter, r *http.Request) {
	text := r.URL.Query().Get("text")
	var season model.MikanSeason
	if !readJSONOrFail(w, r, &season) {
		return
	}
	m := mikan.Search(text, &season)
	fillMikanExists(m)
	ok(w, m)
}

func fillMikanExists(m *model.Mikan) {
	ids := map[string]bool{}
	for _, ani := range config.AniList() {
		if ani != nil && ani.BgmUrl != "" {
			ids[service.GetSubjectIdFromAni(ani)] = true
		}
	}
	for wi := range m.Weeks {
		for ii := range m.Weeks[wi].Items {
			if ids[strconv.Itoa(m.Weeks[wi].Items[ii].BgmId)] {
				m.Weeks[wi].Items[ii].Exists = true
			}
		}
	}
}

// handleMikanGroup processes POST /api/mikanGroup.
func (s *Server) handleMikanGroup(w http.ResponseWriter, r *http.Request) {
	rawURL := r.URL.Query().Get("url")
	if rawURL == "" {
		fail(w, "url 不能为空")
		return
	}
	ok(w, mikan.GetGroups(rawURL))
}

// handleThemoviedbName processes POST /api/getThemoviedbName.
func (s *Server) handleThemoviedbName(w http.ResponseWriter, r *http.Request) {
	var body model.ThemoviedbDTO
	if !readJSONOrFail(w, r, &body) {
		return
	}
	t, err := tmdb.GetByName(body.Title, body.Ova)
	if err != nil {
		fail(w, "获取 TMDB 失败")
		return
	}
	name := tmdb.GetFinalName(t)
	ok(w, &model.ThemoviedbVO{ThemoviedbName: name, Tmdb: t})
}

// handleThemoviedbGroup processes POST /api/getThemoviedbGroup.
func (s *Server) handleThemoviedbGroup(w http.ResponseWriter, r *http.Request) {
	var body model.Ani
	if !readJSONOrFail(w, r, &body) {
		return
	}
	if body.Tmdb == nil || body.Tmdb.ID == 0 {
		fail(w, "TmdbId 或 标题 不能为空")
		return
	}
	groups, err := tmdb.GetTmdbGroup(body.Tmdb)
	if err != nil {
		fail(w, err.Error())
		return
	}
	ok(w, groups)
}

// handleScrape processes POST /api/scrape.
func (s *Server) handleScrape(w http.ResponseWriter, r *http.Request) {
	var body model.Ani
	if !readJSONOrFail(w, r, &body) {
		return
	}
	force, _ := strconv.ParseBool(r.URL.Query().Get("force"))
	go func() {
		_ = scrape.Scrape(&body, force)
	}()
	okMsg(w, "已开始刮削 "+body.Title)
}

// handleBatchScrape processes POST /api/batchScrape.
func (s *Server) handleBatchScrape(w http.ResponseWriter, r *http.Request) {
	var ids []string
	if !readJSONOrFail(w, r, &ids) {
		return
	}
	force, _ := strconv.ParseBool(r.URL.Query().Get("force"))
	go func() {
		for _, ani := range config.AniList() {
			if ani != nil && contains(ids, ani.ID) {
				_ = scrape.Scrape(ani, force)
			}
		}
	}()
	okMsg(w, "已开始刮削")
}

// handleEmbyViews processes POST /api/getEmbyViews.
func (s *Server) handleEmbyViews(w http.ResponseWriter, r *http.Request) {
	var body model.NotificationConfig
	if !readJSONOrFail(w, r, &body) {
		return
	}
	if body.EmbyHost == "" || body.EmbyApiKey == "" {
		fail(w, "emby 未配置完成")
		return
	}
	views := fetchEmbyViews(body.EmbyHost, body.EmbyApiKey)
	ok(w, views)
}

// handleEmbyWebHook processes POST /api/embyWebHook.
func (s *Server) handleEmbyWebHook(w http.ResponseWriter, r *http.Request) {
	var body model.EmbyWebHook
	if !readJSONOrFail(w, r, &body) {
		return
	}
	handleEmbyEvent(&body)
	ok(w, nil)
}