package server

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"ani-rss/internal/config"
	"ani-rss/internal/model"
	"ani-rss/internal/service"
)

// handleConfig processes POST /api/config.
// The login password is blanked before returning (mirrors Java: the frontend
// treats an empty password field as "keep current password"; returning the
// stored MD5 would cause the UI to re-hash it on every save).
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	cfg := *config.Get()
	cfg.Login.Password = ""
	ok(w, &cfg)
}

// handleSetConfig processes POST /api/setConfig.
func (s *Server) handleSetConfig(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		fail(w, err.Error())
		return
	}
	if len(body) == 0 {
		fail(w, "body is empty")
		return
	}
	if err := service.SetConfigRaw(body); err != nil {
		fail(w, err.Error())
		return
	}
	okMsg(w, "修改成功")
}

// handleClearCache processes POST /api/clearCache.
func (s *Server) handleClearCache(w http.ResponseWriter, r *http.Request) {
	msg, err := service.ClearCache()
	if err != nil {
		fail(w, err.Error())
		return
	}
	okMsg(w, msg)
}

// handleTrackersUpdate processes POST /api/trackersUpdate.
func (s *Server) handleTrackersUpdate(w http.ResponseWriter, r *http.Request) {
	var body model.Config
	if !readJSONOrFail(w, r, &body) {
		return
	}
	go service.UpdateTrackers(&body)
	ok(w, nil)
}

// handleTestProxy processes POST /api/testProxy.
func (s *Server) handleTestProxy(w http.ResponseWriter, r *http.Request) {
	rawURL := r.URL.Query().Get("url")
	if rawURL == "" {
		fail(w, "url 不能为空")
		return
	}
	var body model.Config
	if !readJSONOrFail(w, r, &body) {
		return
	}
	test, err := service.TestProxy(rawURL, &body)
	if err != nil {
		fail(w, err.Error())
		return
	}
	ok(w, test)
}

// handleDownloadLoginTest processes POST /api/downloadLoginTest.
func (s *Server) handleDownloadLoginTest(w http.ResponseWriter, r *http.Request) {
	var body model.Config
	if !readJSONOrFail(w, r, &body) {
		return
	}
	if err := service.DownloadLoginTest(); err != nil {
		fail(w, "登录失败")
		return
	}
	okMsg(w, "登录成功")
}

// handlePing handles ANY /api/ping.
func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	ok(w, nil)
}

// handleCustomJs serves GET /api/custom.js.
func (s *Server) handleCustomJs(w http.ResponseWriter, r *http.Request) {
	js := config.Get().CustomJs
	if strings.TrimSpace(js) == "" {
		js = "// empty js"
	}
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Write([]byte(js))
}

// handleCustomCss serves GET /api/custom.css.
func (s *Server) handleCustomCss(w http.ResponseWriter, r *http.Request) {
	css := config.Get().CustomCss
	if strings.TrimSpace(css) == "" {
		css = "/* empty css */"
	}
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Write([]byte(css))
}

// handleExportConfig serves GET /api/exportConfig (zip download).
func (s *Server) handleExportConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `inline; filename="ani-rss.backup."+version+".zip"`)
	_ = service.ExportConfig(w)
}

// handleImportConfig processes POST /api/importConfig (multipart zip).
func (s *Server) handleImportConfig(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		fail(w, err.Error())
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		fail(w, "未获取到文件")
		return
	}
	defer file.Close()
	tmp := filepath.Join(os.TempDir(), "ani-rss-import.zip")
	f, err := os.Create(tmp)
	if err != nil {
		fail(w, err.Error())
		return
	}
	if _, err := io.Copy(f, file); err != nil {
		f.Close()
		fail(w, err.Error())
		return
	}
	f.Close()
	defer os.Remove(tmp)
	if err := service.ImportConfig(tmp); err != nil {
		fail(w, err.Error())
		return
	}
	okMsg(w, "导入成功")
}