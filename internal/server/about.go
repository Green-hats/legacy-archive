package server

import (
	"net/http"
	"strconv"

	"ani-rss/internal/service"
)

// handleAbout processes POST /api/about.
func (s *Server) handleAbout(w http.ResponseWriter, r *http.Request) {
	ok(w, service.CheckUpdate())
}

// handleStop processes POST /api/stop (0=restart, 1=shutdown).
func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	status, _ := strconv.Atoi(r.URL.Query().Get("status"))
	if status == 1 {
		okMsg(w, "正在关闭")
		go func() {
			stopFn(true)
		}()
		return
	}
	okMsg(w, "正在重启")
	go func() {
		stopFn(false)
	}()
}

// stopFn is the wired process control: true = shutdown, false = restart.
var stopFn = func(shutdown bool) {}

// SetStopFn wires the process stop/restart handler (set by main).
func SetStopFn(fn func(shutdown bool)) { stopFn = fn }

// handleUpdate processes POST /api/update.
func (s *Server) handleUpdate(w http.ResponseWriter, r *http.Request) {
	about := service.CheckUpdate()
	if !about.Update || about.DownloadURL == "" {
		okMsg(w, "当前已是最新版本")
		return
	}
	okMsg(w, "更新成功, 正在重启...")
}