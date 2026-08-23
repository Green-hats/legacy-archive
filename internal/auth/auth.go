package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"ani-rss/internal/cache"
	"ani-rss/internal/config"
	"ani-rss/internal/model"
)

var (
	authKeyCache = cache.New(64)
	loginCounts  = cache.New(8192)
	ipv4Re       = regexp.MustCompile(`^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$`)
	cidrRe       = regexp.MustCompile(`^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})/\d{1,2}$`)
	rangeRe      = regexp.MustCompile(`^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})-(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$`)
)

// GetIP mirrors AuthUtil.getIp.
func GetIP(r *http.Request) string {
	cfg := config.Get()
	ip := RemoteAddr(r)
	if !cfg.ReverseProxyTrustIpListEnabled {
		return ip
	}
	if !containsStr(cfg.ReverseProxyTrustIpList, ip) {
		return ip
	}
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		first := strings.TrimSpace(strings.Split(fwd, ",")[0])
		if first != "" {
			return first
		}
	}
	return ip
}

// RemoteAddr extracts the client IP from the request.
func RemoteAddr(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func containsStr(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// prettyLoginJSON reproduces GsonStatic pretty-printed output for a Login:
//
//	{
//	  "username": "...",
//	  "password": "...",
//	  "ip": "...",
//	  "key": "..."
//	}
func prettyLoginJSON(l *model.Login) string {
	return fmt.Sprintf("{\n  \"username\": %q,\n  \"password\": %q,\n  \"ip\": %q,\n  \"key\": %q\n}",
		l.Username, l.Password, l.IP, l.Key)
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// ResetKey generates a new auth key and caches it for loginEffectiveHours.
func ResetKey() string {
	cfg := config.Get()
	key := cfg.UUID
	if cfg.MultiLoginForbidden {
		key = model.NewUUID()
	}
	hours := cfg.LoginEffectiveHours
	if hours <= 0 {
		hours = 3
	}
	authKeyCache.Put("auth_key", key, int64(hours)*int64(time.Hour/time.Millisecond))
	return key
}

// GetAuthKey returns the current cached auth key, resetting if missing.
func GetAuthKey() string {
	if v, ok := authKeyCache.Get("auth_key"); ok {
		return v.(string)
	}
	return ResetKey()
}

// ResetTime refreshes the token TTL (on successful header auth) without
// regenerating the key (mirrors AuthUtil.resetTime).
func ResetTime() {
	if v, ok := authKeyCache.Get("auth_key"); ok {
		cfg := config.Get()
		hours := cfg.LoginEffectiveHours
		if hours <= 0 {
			hours = 3
		}
		authKeyCache.Put("auth_key", v.(string), int64(hours)*int64(time.Hour/time.Millisecond))
	}
}

// GetAuth computes the sha256 auth token for a login.
func GetAuth(l *model.Login) string {
	l.Key = GetAuthKey()
	return sha256Hex(prettyLoginJSON(l))
}

// GetLogin builds the Login used for auth comparison (from stored config).
func GetLogin(r *http.Request) *model.Login {
	cfg := config.Get()
	l := cfg.Login
	if cfg.VerifyLoginIp {
		l.IP = GetIP(r)
	} else {
		l.IP = ""
	}
	return &l
}

// CheckAuth verifies the request against the auth chain: IP whitelist,
// Authorization header token, form `s` token, or API key.
func CheckAuth(r *http.Request) bool {
	if TestIPWhitelist(r) {
		return true
	}
	if TestHeader(r) {
		return true
	}
	if TestForm(r) {
		return true
	}
	if TestApiKey(r) {
		return true
	}
	return false
}

// TestIPWhitelist checks the IP whitelist.
func TestIPWhitelist(r *http.Request) bool {
	cfg := config.Get()
	if !cfg.IpWhitelist || cfg.IpWhitelistStr == "" {
		return false
	}
	ip := GetIP(r)
	if ip == "" {
		return false
	}
	for _, line := range strings.Split(cfg.IpWhitelistStr, "\n") {
		s := strings.TrimSpace(line)
		if s == "" {
			continue
		}
		if s == ip {
			return true
		}
		if !ipv4Re.MatchString(ip) {
			continue
		}
		if strings.Contains(s, "*") {
			if ipWildcardMatch(s, ip) {
				return true
			}
		}
		if cidrRe.MatchString(s) {
			if ipInCIDR(ip, s) {
				return true
			}
		}
		if ipInRange(ip, s) {
			return true
		}
	}
	return false
}

func ipWildcardMatch(pattern, ip string) bool {
	pParts := strings.Split(pattern, ".")
	iParts := strings.Split(ip, ".")
	if len(pParts) != 4 || len(iParts) != 4 {
		return false
	}
	for i := 0; i < 4; i++ {
		if pParts[i] == "*" {
			continue
		}
		if pParts[i] != iParts[i] {
			return false
		}
	}
	return true
}

func ipInCIDR(ip, cidr string) bool {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	return ipNet.Contains(parsed)
}

func ipInRange(ip, s string) bool {
	m := rangeRe.FindStringSubmatch(s)
	if m == nil {
		return false
	}
	start := net.ParseIP(fmt.Sprintf("%s.%s.%s.%s", m[1], m[2], m[3], m[4]))
	end := net.ParseIP(fmt.Sprintf("%s.%s.%s.%s", m[5], m[6], m[7], m[8]))
	cur := net.ParseIP(ip)
	if start == nil || end == nil || cur == nil {
		return false
	}
	return bytesLE(start, cur) && bytesLE(cur, end)
}

func bytesLE(a, b net.IP) bool {
	aa := a.To4()
	bb := b.To4()
	for i := 0; i < 4; i++ {
		if aa[i] < bb[i] {
			return true
		}
		if aa[i] > bb[i] {
			return false
		}
	}
	return true
}

// TestHeader checks the Authorization header token.
func TestHeader(r *http.Request) bool {
	s := r.Header.Get("Authorization")
	if s == "" {
		return false
	}
	if GetAuth(GetLogin(r)) == s {
		ResetTime()
		return true
	}
	return false
}

// TestForm checks the `s` query/form param token.
func TestForm(r *http.Request) bool {
	s := r.URL.Query().Get("s")
	if s == "" {
		_ = r.ParseMultipartForm(0)
		s = r.FormValue("s")
	}
	if s == "" {
		return false
	}
	return GetAuth(GetLogin(r)) == s
}

// TestApiKey checks the api key (header or query param named api-key/x-api-key/s).
func TestApiKey(r *http.Request) bool {
	apiKey := config.Get().ApiKey
	if apiKey == "" {
		return false
	}
	for _, k := range []string{"api-key", "x-api-key", "s"} {
		s := r.Header.Get(k)
		if s == "" {
			s = r.URL.Query().Get(k)
		}
		if s != "" && apiKey == s {
			return true
		}
	}
	return false
}

// LimitLoginAttempts enforces the 30-failure-per-day limit per IP.
// isAdd=true increments the counter; returns true when the IP is now blocked.
func LimitLoginAttempts(ip string, isAdd bool) bool {
	cfg := config.Get()
	if !cfg.LimitLoginAttempts {
		return false
	}
	key := "LimitLoginAttempts#" + ip
	day := int64(24 * time.Hour / time.Millisecond)
	if !loginCounts.Contains(key) {
		if isAdd {
			loginCounts.Put(key, 1, day)
		}
		return false
	}
	v, _ := loginCounts.Get(key)
	count := 0
	if v != nil {
		if n, ok := v.(int); ok {
			count = n
		}
	}
	if isAdd {
		count++
	}
	loginCounts.Put(key, count, day)
	return count >= 30
}

// ClearLimitLoginAttempts removes the login failure counter for an IP.
func ClearLimitLoginAttempts(ip string) {
	loginCounts.Remove("LimitLoginAttempts#" + ip)
}