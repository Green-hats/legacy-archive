package server

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"ani-rss/internal/bgm"
	"ani-rss/internal/config"
	"ani-rss/internal/model"
	"ani-rss/internal/service"
)

// handleAddAni processes POST /api/addAni.
func (s *Server) handleAddAni(w http.ResponseWriter, r *http.Request) {
	var body model.Ani
	if !readJSONOrFail(w, r, &body) {
		return
	}
	if err := service.AddAni(&body); err != nil {
		fail(w, err.Error())
		return
	}
	okMsg(w, "添加订阅成功")
}

// handleSetAni processes POST /api/setAni.
func (s *Server) handleSetAni(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		fail(w, err.Error())
		return
	}
	move, _ := strconv.ParseBool(r.URL.Query().Get("move"))
	if err := service.SetAniRaw(body, move); err != nil {
		fail(w, err.Error())
		return
	}
	okMsg(w, "修改成功")
}

// handleDeleteAni processes POST /api/deleteAni.
func (s *Server) handleDeleteAni(w http.ResponseWriter, r *http.Request) {
	var ids []string
	if !readJSONOrFail(w, r, &ids) {
		return
	}
	deleteFiles, _ := strconv.ParseBool(r.URL.Query().Get("deleteFiles"))
	service.DeleteAni(ids, deleteFiles)
	okMsg(w, "删除订阅成功")
}

// handleListAni processes POST /api/listAni.
func (s *Server) handleListAni(w http.ResponseWriter, r *http.Request) {
	ok(w, service.ListAni())
}

// handleUpdateTotalEpisodeNumber processes POST /api/updateTotalEpisodeNumber.
func (s *Server) handleUpdateTotalEpisodeNumber(w http.ResponseWriter, r *http.Request) {
	var ids []string
	if !readJSONOrFail(w, r, &ids) {
		return
	}
	force, _ := strconv.ParseBool(r.URL.Query().Get("force"))
	go service.UpdateTotalEpisodeNumber(ids, force)
	okMsg(w, "已开始更新总集数")
}

// handleBatchEnable processes POST /api/batchEnable.
func (s *Server) handleBatchEnable(w http.ResponseWriter, r *http.Request) {
	var ids []string
	if !readJSONOrFail(w, r, &ids) {
		return
	}
	value, _ := strconv.ParseBool(r.URL.Query().Get("value"))
	service.BatchEnable(ids, value)
	okMsg(w, "修改完成")
}

// handleRefreshAll processes POST /api/refreshAll.
func (s *Server) handleRefreshAll(w http.ResponseWriter, r *http.Request) {
	service.RefreshAll()
	okMsg(w, "已开始刷新RSS")
}

// handleRefreshAni processes POST /api/refreshAni.
func (s *Server) handleRefreshAni(w http.ResponseWriter, r *http.Request) {
	var body model.IdDTO
	if !readJSONOrFail(w, r, &body) {
		return
	}
	for _, ani := range config.AniList() {
		if ani != nil && ani.ID == body.ID {
			service.RefreshAni(ani)
			break
		}
	}
	okMsg(w, "已开始刷新RSS")
}

// handleRssToAni processes POST /api/rssToAni.
func (s *Server) handleRssToAni(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		fail(w, err.Error())
		return
	}
	var dto model.RssToAniDTO
	if err := json.Unmarshal(body, &dto); err != nil {
		fail(w, err.Error())
		return
	}
	// Java defaults enable to true when the field is absent (the frontend's
	// bulk-add flow never sends `enable`). Respect an explicit false.
	var raw map[string]json.RawMessage
	_ = json.Unmarshal(body, &raw)
	if _, ok := raw["enable"]; !ok {
		dto.Enable = true
	}
	ani, err := service.RssToAni(&dto)
	if err != nil {
		fail(w, "RSS解析失败 "+err.Error())
		return
	}
	ok(w, ani)
}

// handlePreviewAni processes POST /api/previewAni.
func (s *Server) handlePreviewAni(w http.ResponseWriter, r *http.Request) {
	var body model.Ani
	if !readJSONOrFail(w, r, &body) {
		return
	}
	ok(w, service.PreviewAni(&body))
}

// handleDownloadPath processes POST /api/downloadPath.
func (s *Server) handleDownloadPath(w http.ResponseWriter, r *http.Request) {
	var body model.Ani
	if !readJSONOrFail(w, r, &body) {
		return
	}
	ok(w, service.DownloadPathPreview(&body))
}

// handleImportAni processes POST /api/importAni.
func (s *Server) handleImportAni(w http.ResponseWriter, r *http.Request) {
	var body model.ImportAniDataDTO
	if !readJSONOrFail(w, r, &body) {
		return
	}
	for _, ani := range body.AniList {
		if ani == nil {
			continue
		}
		if body.Conflict == "SKIP" {
			exists := false
			for _, a := range config.AniList() {
				if a != nil && a.ID == ani.ID {
					exists = true
					break
				}
			}
			if exists {
				continue
			}
		}
		_ = service.AddAni(ani)
	}
	okMsg(w, "导入成功")
}

// handleRefreshCover processes POST /api/refreshCover.
func (s *Server) handleRefreshCover(w http.ResponseWriter, r *http.Request) {
	var body model.Ani
	if !readJSONOrFail(w, r, &body) {
		return
	}
	path := saveCover(body.Image)
	ok(w, path)
}

var _ = bgm.GetBgmInfo