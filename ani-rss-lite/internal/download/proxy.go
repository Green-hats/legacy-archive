package download

import (
	"io"
	"net/http"
	"strings"

	"ani-rss/internal/util"
)

// ProxyCloudFile streams a cloud file through w instead of redirecting, so
// external players (MPV, IINA, ...) only talk to the local server. 115 CDN
// links are bound to the User-Agent used to request them, so the proxy supplies
// the same browser UA and forwards Range requests for seekable playback.
func ProxyCloudFile(w http.ResponseWriter, r *http.Request, cloudPath string) {
	cloud, ok := Type().(CloudClient)
	if !ok {
		http.NotFound(w, r)
		return
	}
	rawURL := cloud.FileURL(cloudPath)
	if rawURL == "" {
		http.NotFound(w, r)
		return
	}
	method := http.MethodGet
	if r.Method == http.MethodHead {
		method = http.MethodHead
	}
	req, err := http.NewRequest(method, rawURL, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	if rng := r.Header.Get("Range"); rng != "" {
		req.Header.Set("Range", rng)
	}
	if _, ok := cloud.(*Pan115); ok {
		req.Header.Set("User-Agent", ua115)
	}
	resp, err := util.StreamClient().Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// prefer a proper media type for the file so browsers can play it,
	// instead of echoing the CDN's generic application/octet-stream.
	if mt := util.VideoMimeType(cloudPath); mt != "" {
		w.Header().Set("Content-Type", mt)
	} else if ct := resp.Header.Get("Content-Type"); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	if cr := resp.Header.Get("Content-Range"); cr != "" {
		w.Header().Set("Content-Range", cr)
	}
	if l := resp.Header.Get("Content-Length"); l != "" && !strings.EqualFold(l, "0") {
		w.Header().Set("Content-Length", l)
	}
	if ar := resp.Header.Get("Accept-Ranges"); ar != "" {
		w.Header().Set("Accept-Ranges", ar)
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(resp.StatusCode)
	if r.Method != http.MethodHead {
		_, _ = io.Copy(w, resp.Body)
	}
}