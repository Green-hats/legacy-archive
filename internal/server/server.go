package server

import (
	"encoding/json"
	"net/http"

	"ani-rss/internal/auth"
	"ani-rss/internal/config"
	"ani-rss/internal/model"
)

// Server is the HTTP application server.
type Server struct {
	Router *Router
}

// New builds a server with all routes registered.
func New() *Server {
	s := &Server{Router: NewRouter()}
	s.register()
	return s
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	applyCORS(w, r)
	s.Router.ServeHTTP(w, r)
}

func applyCORS(w http.ResponseWriter, r *http.Request) {
	if config.Get().AllowCors {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Max-Age", "0")
	}
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	b, _ := json.Marshal(v)
	w.Write(b)
}

// writeResult writes a model.Result with HTTP 200 (Java always returns 200 for JSON).
func writeResult(w http.ResponseWriter, res *model.Result) {
	writeJSON(w, http.StatusOK, res)
}

// ok writes a success result.
func ok(w http.ResponseWriter, data interface{}) {
	writeResult(w, model.NewResult(data))
}

// okMsg writes a success result with a custom message.
func okMsg(w http.ResponseWriter, msg string) {
	writeResult(w, model.NewMessage(msg))
}

// fail writes an error result (code 500).
func fail(w http.ResponseWriter, msg string) {
	writeResult(w, model.NewError(msg))
}

// requireAuth wraps a handler with auth checking. Failed auth returns 403 登录已失效.
func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !auth.CheckAuth(r) {
			writeResult(w, model.NewResultCode(403, "登录已失效"))
			return
		}
		next(w, r)
	}
}

// readJSON decodes a JSON request body into v.
func readJSON(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	return dec.Decode(v)
}

// readJSONOrFail decodes the body, writing a 500 error on failure.
func readJSONOrFail(w http.ResponseWriter, r *http.Request, v interface{}) bool {
	if err := readJSON(r, v); err != nil {
		fail(w, err.Error())
		return false
	}
	return true
}