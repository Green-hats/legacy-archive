package server

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"ani-rss/internal/auth"
	"ani-rss/internal/config"
	"ani-rss/internal/log"
	"ani-rss/internal/model"
)

// handleLogin processes POST /api/login.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	ip := auth.GetIP(r)
	if auth.LimitLoginAttempts(ip, false) {
		writeResult(w, model.NewResultCode(403, fmt.Sprintf("失败次数过多, 已限制登录 %s", ip)))
		return
	}
	var myLogin model.Login
	if !readJSONOrFail(w, r, &myLogin) {
		return
	}
	if myLogin.Username == "" || myLogin.Password == "" {
		fail(w, "用户名不能为空")
		return
	}
	cfg := config.Get()
	stored := cfg.Login
	if cfg.VerifyLoginIp {
		myLogin.IP = ip
	} else {
		myLogin.IP = ""
	}
	if stored.Username == myLogin.Username && stored.Password == myLogin.Password {
		auth.ResetKey()
		auth.ClearLimitLoginAttempts(ip)
		log.Infof("login", "登录成功 %s ip: %s", stored.Username, ip)
		token := auth.GetAuth(&myLogin)
		writeResult(w, &model.Result{Code: 200, Message: "登录成功", Data: token, T: model.Now().UnixMilli()})
		return
	}
	auth.LimitLoginAttempts(ip, true)
	log.Warnf("login", "登陆失败 %s ip: %s", myLogin.Username, ip)
	time.Sleep(time.Duration(500+rand.Intn(4500)) * time.Millisecond)
	fail(w, "用户名或密码错误")
}

// handleTestIpWhitelist processes POST /api/testIpWhitelist.
func (s *Server) handleTestIpWhitelist(w http.ResponseWriter, r *http.Request) {
	if auth.TestIPWhitelist(r) {
		ok(w, nil)
		return
	}
	fail(w, "IP 不在白名单内")
}