package util

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"ani-rss/internal/config"
	"ani-rss/internal/model"
)

// Version is the application version (overridden at build time).
var Version = "0.1.0"

// DefaultClient returns an HTTP client honoring config proxy settings.
func DefaultClient() *http.Client {
	return newClient(config.Get(), 20)
}

// ClientFor returns an HTTP client with the given timeout (seconds), honoring proxy.
func ClientFor(timeoutSec int) *http.Client {
	return newClient(config.Get(), timeoutSec)
}

// StreamClient returns an HTTP client without a total request timeout, for
// streaming large bodies (e.g. cloud media), honoring config proxy settings.
func StreamClient() *http.Client {
	c := newClient(config.Get(), 20)
	c.Timeout = 0
	return c
}

func newClient(cfg *model.Config, timeoutSec int) *http.Client {
	if cfg == nil {
		cfg = &model.Config{}
	}
	if timeoutSec <= 0 {
		timeoutSec = 20
	}
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout: 10 * time.Second,
		}).DialContext,
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	if cfg.Proxy {
		if p := proxyURL(cfg); p != nil {
			transport.Proxy = http.ProxyURL(p)
		}
	}
	return &http.Client{
		Transport: transport,
		Timeout:   time.Duration(timeoutSec) * time.Second,
	}
}

func proxyURL(cfg *model.Config) *url.URL {
	if cfg.ProxyHost == "" {
		return nil
	}
	scheme := "http"
	if strings.HasPrefix(cfg.ProxyHost, "http://") || strings.HasPrefix(cfg.ProxyHost, "https://") || strings.HasPrefix(cfg.ProxyHost, "socks5://") {
		scheme = ""
	}
	raw := cfg.ProxyHost
	if scheme != "" {
		raw = scheme + "://" + raw
	}
	if cfg.ProxyUsername != "" || cfg.ProxyPassword != "" {
		u, err := url.Parse(raw)
		if err != nil {
			return nil
		}
		if cfg.ProxyUsername != "" {
			u.User = url.UserPassword(cfg.ProxyUsername, cfg.ProxyPassword)
		}
		return u
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil
	}
	return u
}

// UserAgent returns the standard UA string.
func UserAgent() string {
	return "ani-rss-go/" + Version
}

// Get fetches a URL with the config UA, returning the response.
func Get(rawURL string) (*http.Response, error) {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent())
	return DefaultClient().Do(req)
}

// GetBytes fetches a URL and returns the body bytes.
func GetBytes(rawURL string) ([]byte, error) {
	resp, err := Get(rawURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.New("http status " + resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// MD5Hex computes the md5 digest hex of a string.
func MD5Hex(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}