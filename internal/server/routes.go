package server

import "net/http"

// register wires all API routes. Paths are matched case-insensitively.
func (s *Server) register() {
	r := s.Router

	// no-auth endpoints
	r.Handle(http.MethodPost, "/api/login", s.handleLogin)
	r.Handle(http.MethodPost, "/api/testIpWhitelist", s.handleTestIpWhitelist)
	r.Handle("ANY", "/api/ping", s.handlePing)
	r.Handle(http.MethodGet, "/api/custom.js", s.handleCustomJs)
	r.Handle(http.MethodGet, "/api/custom.css", s.handleCustomCss)

	// config
	r.Handle(http.MethodPost, "/api/config", requireAuth(s.handleConfig))
	r.Handle(http.MethodPost, "/api/setConfig", requireAuth(s.handleSetConfig))
	r.Handle(http.MethodPost, "/api/clearCache", requireAuth(s.handleClearCache))
	r.Handle(http.MethodPost, "/api/trackersUpdate", requireAuth(s.handleTrackersUpdate))
	r.Handle(http.MethodPost, "/api/testProxy", requireAuth(s.handleTestProxy))
	r.Handle(http.MethodPost, "/api/downloadLoginTest", requireAuth(s.handleDownloadLoginTest))
	r.Handle(http.MethodGet, "/api/exportConfig", requireAuth(s.handleExportConfig))
	r.Handle(http.MethodPost, "/api/importConfig", requireAuth(s.handleImportConfig))

	// about
	r.Handle(http.MethodPost, "/api/about", requireAuth(s.handleAbout))
	r.Handle(http.MethodPost, "/api/stop", requireAuth(s.handleStop))

	// ani
	r.Handle(http.MethodPost, "/api/addAni", requireAuth(s.handleAddAni))
	r.Handle(http.MethodPost, "/api/setAni", requireAuth(s.handleSetAni))
	r.Handle(http.MethodPost, "/api/deleteAni", requireAuth(s.handleDeleteAni))
	r.Handle(http.MethodPost, "/api/listAni", requireAuth(s.handleListAni))
	r.Handle(http.MethodPost, "/api/updateTotalEpisodeNumber", requireAuth(s.handleUpdateTotalEpisodeNumber))
	r.Handle(http.MethodPost, "/api/batchEnable", requireAuth(s.handleBatchEnable))
	r.Handle(http.MethodPost, "/api/refreshAll", requireAuth(s.handleRefreshAll))
	r.Handle(http.MethodPost, "/api/refreshAni", requireAuth(s.handleRefreshAni))
	r.Handle(http.MethodPost, "/api/rssToAni", requireAuth(s.handleRssToAni))
	r.Handle(http.MethodPost, "/api/previewAni", requireAuth(s.handlePreviewAni))
	r.Handle(http.MethodPost, "/api/downloadPath", requireAuth(s.handleDownloadPath))
	r.Handle(http.MethodPost, "/api/importAni", requireAuth(s.handleImportAni))
	r.Handle(http.MethodPost, "/api/refreshCover", requireAuth(s.handleRefreshCover))

	// logs
	r.Handle(http.MethodPost, "/api/logs", requireAuth(s.handleLogs))
	r.Handle(http.MethodPost, "/api/clearLogs", requireAuth(s.handleClearLogs))
	r.Handle(http.MethodGet, "/api/downloadLogs", requireAuth(s.handleDownloadLogs))

	// notification
	r.Handle(http.MethodPost, "/api/testNotification", requireAuth(s.handleTestNotification))
	r.Handle(http.MethodPost, "/api/newNotification", requireAuth(s.handleNewNotification))
	r.Handle(http.MethodPost, "/api/getTgUpdates", requireAuth(s.handleGetTgUpdates))

	// file / play / proxy / torrents
	r.Handle(http.MethodGet, "/api/file", requireAuth(s.handleFile))
	r.Handle(http.MethodGet, "/api/calendar.ics", requireAuth(s.handleCalendarIcs))
	r.Handle(http.MethodGet, "/api/proxyImage", requireAuth(s.handleProxyImage))
	r.Handle(http.MethodPost, "/api/torrentsInfos", requireAuth(s.handleTorrentsInfos))
	r.Handle(http.MethodPost, "/api/deleteTorrent", requireAuth(s.handleDeleteTorrent))
	r.Handle(http.MethodPost, "/api/playList", requireAuth(s.handlePlayList))
	r.Handle(http.MethodPost, "/api/upload", requireAuth(s.handleUpload))

	// bgm
	r.Handle(http.MethodPost, "/api/searchBgm", requireAuth(s.handleSearchBgm))
	r.Handle(http.MethodPost, "/api/getAniBySubjectId", requireAuth(s.handleGetAniBySubjectId))
	r.Handle(http.MethodPost, "/api/getBgmTitle", requireAuth(s.handleGetBgmTitle))
	r.Handle(http.MethodPost, "/api/rate", requireAuth(s.handleRate))
	r.Handle(http.MethodPost, "/api/setRate", requireAuth(s.handleSetRate))
	r.Handle(http.MethodPost, "/api/meBgm", requireAuth(s.handleMeBgm))
	r.Handle(http.MethodPost, "/api/bgm/oauth/callback", requireAuth(s.handleBgmOauthCallback))

	// mikan / tmdb / scrape / emby
	r.Handle(http.MethodPost, "/api/mikan", requireAuth(s.handleMikan))
	r.Handle(http.MethodPost, "/api/mikanGroup", requireAuth(s.handleMikanGroup))
	r.Handle(http.MethodPost, "/api/getThemoviedbName", requireAuth(s.handleThemoviedbName))
	r.Handle(http.MethodPost, "/api/getThemoviedbGroup", requireAuth(s.handleThemoviedbGroup))
	r.Handle(http.MethodPost, "/api/scrape", requireAuth(s.handleScrape))
	r.Handle(http.MethodPost, "/api/batchScrape", requireAuth(s.handleBatchScrape))
	r.Handle(http.MethodPost, "/api/getEmbyViews", requireAuth(s.handleEmbyViews))
	r.Handle(http.MethodPost, "/api/embyWebHook", requireAuth(s.handleEmbyWebHook))

	// ani-bt / anime-garden / collection / afdian
	r.Handle(http.MethodPost, "/api/aniBT", requireAuth(s.handleAniBT))
	r.Handle(http.MethodPost, "/api/aniBTGroup", requireAuth(s.handleAniBTGroup))
	r.Handle(http.MethodPost, "/api/animeGardenList", requireAuth(s.handleAnimeGardenList))
	r.Handle(http.MethodPost, "/api/animeGardenGroup", requireAuth(s.handleAnimeGardenGroup))

	// static webui with SPA fallback
	r.NotFound = staticFileServer()
}