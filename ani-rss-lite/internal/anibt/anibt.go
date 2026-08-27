package anibt

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"ani-rss/internal/bgm"
	"ani-rss/internal/config"
	"ani-rss/internal/groupregex"
	"ani-rss/internal/model"
	"ani-rss/internal/util"
)

const host = "https://anibt.net"

func apiGet(path string, params url.Values) ([]byte, error) {
	u := host + path
	if params != nil {
		u += "?" + params.Encode()
	}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", util.UserAgent())
	resp, err := util.DefaultClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.New("anibt http " + resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// List searches ani-bt anime grouped by weekday (mirrors AniBTService.list).
func List(dto *model.AniBTQueryDTO) *model.AniBT {
	title := strings.TrimSpace(dto.Title)
	params := url.Values{}
	if title == "" {
		params.Set("season", dto.Season)
		params.Set("bgmId", bgm.GetSubjectIdByUrl(dto.BgmUrl))
		params.Set("query", "")
	} else {
		params.Set("season", "")
		params.Set("bgmId", "")
		params.Set("query", title)
	}

	b, err := apiGet("/api/seasons/anime", params)
	if err != nil {
		util.LogWarn("anibt", "获取 ani-bt 列表失败 %v", err)
		return &model.AniBT{ByWeekday: []model.AniBTByWeekday{}}
	}
	var resp struct {
		Data model.AniBT `json:"data"`
	}
	if err := json.Unmarshal(b, &resp); err != nil {
		util.LogWarn("anibt", "解析 ani-bt 列表失败 %v", err)
		return &model.AniBT{ByWeekday: []model.AniBTByWeekday{}}
	}
	aniBT := resp.Data

	// subscriptions' bgm ids
	bgmIdSet := map[string]bool{}
	for _, ani := range config.AniList() {
		if ani != nil && ani.BgmUrl != "" {
			bgmIdSet[bgm.GetSubjectIdByUrl(ani.BgmUrl)] = true
		}
	}

	var kept []model.AniBTByWeekday
	for _, wd := range aniBT.ByWeekday {
		var animes []model.AniBTAnime
		for _, a := range wd.Animes {
			if title == "" && a.RssReleaseCount <= 0 {
				continue
			}
			a.Exists = bgmIdSet[string(a.BgmId)]
			animes = append(animes, a)
		}
		sort.SliceStable(animes, func(i, j int) bool { return animes[i].Rating > animes[j].Rating })
		if len(animes) > 0 {
			wd.Animes = animes
			kept = append(kept, wd)
		}
	}
	sort.SliceStable(kept, func(i, j int) bool {
		return util.WeekSortIndex(kept[i].WeekdayLabel) < util.WeekSortIndex(kept[j].WeekdayLabel)
	})
	aniBT.ByWeekday = kept
	return &aniBT
}

// GetGroups fetches the subgroup list for a bgm id.
func GetGroups(bgmId string) []model.AniBTGroup {
	params := url.Values{}
	params.Set("bgmId", bgmId)
	b, err := apiGet("/api/anime/groups", params)
	if err != nil {
		util.LogWarn("anibt", "获取 ani-bt 字幕组失败 %v", err)
		return nil
	}
	var resp struct {
		Data struct {
			Groups []model.AniBTGroup `json:"groups"`
		} `json:"data"`
	}
	if err := json.Unmarshal(b, &resp); err != nil {
		util.LogWarn("anibt", "解析 ani-bt 字幕组失败 %v", err)
		return nil
	}
	for i := range resp.Data.Groups {
		g := &resp.Data.Groups[i]
		g.Rss = fmt.Sprintf("https://anibt.net/rss/anime.xml?bgmId=%s&groupSlug=%s", bgmId, g.Slug)
		var titles []string
		for _, it := range g.Items {
			titles = append(titles, it.Title)
		}
		g.GroupRegex = groupregex.ToGroupRegex(titles)
		for j := range g.Items {
			g.Items[j].FormatSize = util.FormatSize(g.Items[j].Size)
		}
		g.BgmId = model.StrID(bgmId)
	}
	return resp.Data.Groups
}