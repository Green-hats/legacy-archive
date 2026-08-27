package server

import (
	"net/http"
	"strconv"

	"ani-rss/internal/model"
	"ani-rss/internal/util"
)

// handleAbout processes POST /api/about (版本信息,不检查更新).
func (s *Server) handleAbout(w http.ResponseWriter, r *http.Request) {
	ok(w, &model.About{
		Version: util.Version,
		Date:    model.DateTime(model.Now()),
	})
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