package rss

import (
	"encoding/xml"
	"errors"
	"io"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"ani-rss/internal/config"
	"ani-rss/internal/model"
	"ani-rss/internal/rename"
	"ani-rss/internal/util"
)

var (
	regMagnet    = regexp.MustCompile(`^magnet:\?xt=urn:btih:(\w+)`)
	regEd2k      = regexp.MustCompile(`^ed2k://\|file\|([^|]+)\|(\d+)\|([A-Fa-f0-9]{32})\|/$`)
	regGuidHex   = regexp.MustCompile(`^([a-z]|[0-9])+$`)
	regSubgroup  = regexp.MustCompile(`^\{\{(.+)}}:(.+)$`)
)

// xmlItem is a raw RSS item.
type xmlItem struct {
	Title      string    `xml:"title"`
	Link       string    `xml:"link"`
	GUID       string    `xml:"guid"`
	PubDate    string    `xml:"pubDate"`
	Enclosure  *enclosure `xml:"enclosure"`
	NyaaSize   string    `xml:"nyaa:size"`
	NyaaHash   string    `xml:"nyaa:infoHash"`
	Torrent    *xmlTorrent `xml:"torrent"`
}

type enclosure struct {
	URL    string `xml:"url,attr"`
	Length string `xml:"length,attr"`
}

type xmlTorrent struct {
	InfoHash string `xml:"infohash"`
	PubDate  string `xml:"pubDate"`
}

type rssChannel struct {
	Items []xmlItem `xml:"item"`
}

type rssDocument struct {
	Channel rssChannel `xml:"channel"`
}

// GetRSS fetches and validates the RSS XML body.
func GetRSS(rawURL string) (string, error) {
	cfg := config.Get()
	timeout := cfg.RssTimeout
	if timeout <= 0 {
		timeout = 20
	}
	rawURL = normalizeURL(rawURL)
	req, err := util.NewRequest("GET", rawURL)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", util.UserAgent())
	resp, err := util.ClientFor(timeout).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", errors.New("http status " + resp.Status)
	}
	ct := resp.Header.Get("Content-Type")
	ct = strings.ToLower(ct)
	if ct != "" && !strings.Contains(ct, "xml") {
		return "", errors.New("content type not xml: " + ct)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	body := strings.TrimSpace(string(b))
	if !strings.HasPrefix(body, "<") {
		return "", errors.New("xml error")
	}
	return body, nil
}

// normalizeURL percent-encodes the query so that unencoded characters in
// user-entered RSS urls (e.g. Chinese fansub names) are sent correctly.
func normalizeURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	if q := u.Query(); len(q) > 0 {
		u.RawQuery = q.Encode()
		return u.String()
	}
	return rawURL
}

// GetItems aggregates main + standby RSS items for an ani, sorted by episode.
func GetItems(ani *model.Ani) []*model.Item {
	cfg := config.Get()
	subgroup := ani.Subgroup
	if strings.TrimSpace(subgroup) == "" {
		subgroup = "未知字幕组"
	}
	var items []*model.Item
	for _, it := range getItems(ani, ani.URL, subgroup) {
		it.Master = true
		items = append(items, it)
	}

	if !cfg.StandbyRss {
		sort.SliceStable(items, func(i, j int) bool { return items[i].Episode < items[j].Episode })
		return items
	}

	for _, sr := range ani.StandbyRssList {
		time.Sleep(time.Second)
		subgroup = sr.Label
		if strings.TrimSpace(subgroup) == "" {
			subgroup = "未知字幕组"
		}
		clone := ani.Clone()
		clone.Offset = sr.Offset
		for _, it := range getItems(clone, sr.URL, subgroup) {
			it.Master = false
			items = append(items, it)
		}
	}

	items = distinctItems(items, cfg.Coexist)
	sort.SliceStable(items, func(i, j int) bool { return items[i].Episode < items[j].Episode })
	return items
}

func distinctItems(items []*model.Item, coexist bool) []*model.Item {
	seen := map[string]bool{}
	var out []*model.Item
	for _, it := range items {
		var key string
		if coexist {
			key = it.ReName
		} else {
			key = strconv.FormatFloat(it.Episode, 'f', -1, 64)
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, it)
	}
	return out
}

// getItems parses one RSS feed into items (newest first), filtering and renaming.
func getItems(ani *model.Ani, rssURL, subgroupName string) []*model.Item {
	xmlBody, err := GetRSS(rssURL)
	if err != nil {
		util.LogWarn("RSS", "获取RSS失败 %s %v", rssURL, err)
		return nil
	}
	var doc rssDocument
	if err := xml.Unmarshal([]byte(xmlBody), &doc); err != nil {
		util.LogWarn("RSS", "解析RSS失败 %s %v", rssURL, err)
		return nil
	}
	cfg := config.Get()
	globalExcludeList := cfg.Exclude

	var items []*model.Item
	for i := len(doc.Channel.Items) - 1; i >= 0; i-- {
		raw := doc.Channel.Items[i]
		itemTitle := raw.Title
		torrent := ""
		length := "1"
		infoHash := ""
		formatSize := "0MiB"
		var pubDate time.Time

		if raw.Enclosure != nil {
			torrent = raw.Enclosure.URL
			if raw.Enclosure.Length != "" {
				length = raw.Enclosure.Length
			}
			if m := regMagnet.FindStringSubmatch(torrent); len(m) > 1 {
				infoHash = m[1]
			}
			if m := regEd2k.FindStringSubmatch(torrent); len(m) > 3 {
				infoHash = m[3]
			}
		}
		if regGuidHex.MatchString(raw.GUID) {
			infoHash = raw.GUID
		}
		if raw.NyaaHash != "" {
			infoHash = raw.NyaaHash
		}
		if raw.NyaaSize != "" {
			formatSize = raw.NyaaSize
		}
		if t, err := parseDate(raw.PubDate); err == nil {
			pubDate = t
		}
		if raw.Torrent != nil {
			if raw.Torrent.InfoHash != "" {
				infoHash = raw.Torrent.InfoHash
			}
			if pubDate.IsZero() {
				if t, err := parseDate(raw.Torrent.PubDate); err == nil {
					pubDate = t
				}
			}
		}
		if strings.HasSuffix(raw.Link, ".torrent") {
			torrent = raw.Link
		}

		if torrent == "" {
			continue
		}
		if infoHash == "" {
			infoHash = baseName(torrent)
		}
		infoHash = strings.ToLower(infoHash)
		if dec, err := url.PathUnescape(infoHash); err == nil {
			infoHash = dec
		}

		if formatSize == "0MiB" {
			if n := parseInt64Safe(length); n > 0 {
				formatSize = util.FormatSize(n)
			}
		}

		it := &model.Item{
			Subgroup:  subgroupName,
			Episode:   1.0,
			Title:     itemTitle,
			ReName:    itemTitle,
			Torrent:   torrent,
			InfoHash:  infoHash,
			FormatSize: formatSize,
		}
		if !pubDate.IsZero() {
			it.PubDate = model.DateTime(pubDate)
		}

		mapFn := func(s string) string {
			m := regSubgroup.FindStringSubmatch(s)
			if len(m) < 3 {
				return s
			}
			if m[1] == subgroupName {
				return m[2]
			}
			return ""
		}

		if len(ani.Exclude) > 0 {
			drop := false
			for _, ex := range ani.Exclude {
				mapped := mapFn(ex)
				if mapped == "" {
					continue
				}
				if regexpMatch(mapped, it.Title) {
					drop = true
					break
				}
			}
			if drop {
				continue
			}
		}
		if len(ani.Match) > 0 {
			drop := false
			for _, mt := range ani.Match {
				mapped := mapFn(mt)
				if mapped == "" {
					continue
				}
				if !regexpMatch(mapped, it.Title) {
					drop = true
					break
				}
			}
			if drop {
				continue
			}
		}
		if ani.GlobalExclude {
			drop := false
			for _, ex := range globalExcludeList {
				mapped := mapFn(ex)
				if mapped == "" {
					continue
				}
				if regexpMatch(mapped, it.Title) {
					drop = true
					break
				}
			}
			if drop {
				continue
			}
		}
		items = append(items, it)
	}

	var filtered []*model.Item
	for _, it := range items {
		if rename.Rename(ani, it) {
			filtered = append(filtered, it)
		}
	}
	filtered = distinctByEpisodeKeepLast(filtered)
	return filtered
}

func distinctByEpisodeKeepLast(items []*model.Item) []*model.Item {
	m := map[string]*model.Item{}
	var order []string
	for _, it := range items {
		k := strconv.FormatFloat(it.Episode, 'f', -1, 64)
		if _, ok := m[k]; !ok {
			order = append(order, k)
		}
		m[k] = it
	}
	var out []*model.Item
	for _, k := range order {
		out = append(out, m[k])
	}
	return out
}

func regexpMatch(pattern, s string) bool {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return strings.Contains(s, pattern)
	}
	return re.MatchString(s)
}

func parseInt64Safe(s string) int64 {
	var n int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int64(c-'0')
	}
	return n
}

func parseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, errors.New("empty")
	}
	for _, layout := range []string{
		time.RFC1123Z, time.RFC1123, time.RFC822, time.RFC822Z,
		"2006-01-02 15:04:05", "2006-01-02", time.RFC3339,
	} {
		if t, err := time.ParseInLocation(layout, s, model.Loc()); err == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.New("parse failed")
}

func baseName(torrent string) string {
	if idx := strings.LastIndexAny(torrent, "/\\"); idx >= 0 {
		return torrent[idx+1:]
	}
	return torrent
}