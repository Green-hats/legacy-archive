package server

import (
	"net/http"

	"ani-rss/internal/anibt"
	"ani-rss/internal/animegarden"
	"ani-rss/internal/model"
)

// handleAniBT processes POST /api/aniBT (search).
func (s *Server) handleAniBT(w http.ResponseWriter, r *http.Request) {
	var body model.AniBTQueryDTO
	if !readJSONOrFail(w, r, &body) {
		return
	}
	ok(w, anibt.List(&body))
}

// handleAniBTGroup processes POST /api/aniBTGroup.
func (s *Server) handleAniBTGroup(w http.ResponseWriter, r *http.Request) {
	bgmId := r.URL.Query().Get("bgmId")
	if bgmId == "" {
		fail(w, "bgmId 不能为空")
		return
	}
	ok(w, anibt.GetGroups(bgmId))
}

// handleAnimeGardenList processes POST /api/animeGardenList.
func (s *Server) handleAnimeGardenList(w http.ResponseWriter, r *http.Request) {
	bgmUrl := r.URL.Query().Get("bgmUrl")
	ok(w, animegarden.List(bgmUrl))
}

// handleAnimeGardenGroup processes POST /api/animeGardenGroup.
func (s *Server) handleAnimeGardenGroup(w http.ResponseWriter, r *http.Request) {
	bgmId := r.URL.Query().Get("bgmId")
	if bgmId == "" {
		fail(w, "bgmId 不能为空")
		return
	}
	ok(w, animegarden.Group(bgmId))
}