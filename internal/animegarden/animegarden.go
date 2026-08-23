package animegarden

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"ani-rss/internal/bgm"
	"ani-rss/internal/config"
	"ani-rss/internal/groupregex"
	"ani-rss/internal/model"
	"ani-rss/internal/util"
)

const host = "https://api.animes.garden"

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
		return nil, errors.New("animegarden http " + resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// List returns the weekly anime-garden subject list, or a single "搜索" week
// when a bgm url is provided (mirrors AnimeGardenService.list).
func List(bgmUrl string) []model.AnimeGarden {
	if strings.TrimSpace(bgmUrl) != "" {
		bgmId := bgm.GetSubjectIdByUrl(bgmUrl)
		subject := &model.AnimeGardenSubject{ID: model.StrID(bgmId), Exists: true}
		if info, err := bgm.GetBgmInfo(bgmId); err == nil && info != nil {
			subject.Name = bgm.GetFinalName(info)
			subject.Cover = info.Images.Small
		}
		return []model.AnimeGarden{{
			WeekLabel: "搜索",
			Subjects:  []model.AnimeGardenSubject{*subject},
		}}
	}

	b, err := apiGet("/subjects", nil)
	if err != nil {
		util.LogWarn("animegarden", "获取番剧列表失败 %v", err)
		return nil
	}
	var resp struct {
		Subjects []model.AnimeGardenSubject `json:"subjects"`
	}
	if err := json.Unmarshal(b, &resp); err != nil {
		util.LogWarn("animegarden", "解析番剧列表失败 %v", err)
		return nil
	}

	bgmIdSet := map[string]bool{}
	for _, ani := range config.AniList() {
		if ani != nil && ani.BgmUrl != "" {
			bgmIdSet[bgm.GetSubjectIdByUrl(ani.BgmUrl)] = true
		}
	}

	weeks := []string{"星期日", "星期一", "星期二", "星期三", "星期四", "星期五", "星期六"}
	byWeek := map[string][]model.AnimeGardenSubject{}
	for i := range resp.Subjects {
		s := resp.Subjects[i]
		s.Exists = bgmIdSet[string(s.ID)]
		week := weeks[weekdayIndex(s.ActivedAt.Time())]
		s.WeekLabel = week
		byWeek[week] = append(byWeek[week], s)
	}

	var out []model.AnimeGarden
	for _, w := range weeks {
		subs, ok := byWeek[w]
		if !ok {
			continue
		}
		out = append(out, model.AnimeGarden{WeekLabel: w, Subjects: subs})
	}
	sort.SliceStable(out, func(i, j int) bool {
		return util.WeekSortIndex(out[i].WeekLabel) < util.WeekSortIndex(out[j].WeekLabel)
	})
	return out
}

func weekdayIndex(t time.Time) int {
	if t.IsZero() {
		return int(model.Now().Weekday())
	}
	return int(t.Weekday())
}

// Group returns the fansub groups + items for a bgm id.
func Group(bgmId string) []model.AnimeGardenGroup {
	params := url.Values{}
	params.Set("subject", bgmId)
	params.Set("pageSize", "200")
	params.Set("duplicate", "false")
	b, err := apiGet("/resources", params)
	if err != nil {
		util.LogWarn("animegarden", "获取字幕组资源失败 %v", err)
		return nil
	}
	var resp struct {
		Resources []model.AnimeGardenItem `json:"resources"`
	}
	if err := json.Unmarshal(b, &resp); err != nil {
		util.LogWarn("animegarden", "解析字幕组资源失败 %v", err)
		return nil
	}

	var items []model.AnimeGardenItem
	for _, it := range resp.Resources {
		if it.Fansub == nil {
			continue
		}
		it.FormatSize = util.FormatSize(it.Size)
		items = append(items, it)
	}

	groupByFansub := map[string][]model.AnimeGardenItem{}
	for _, it := range items {
		groupByFansub[string(it.Fansub.ID)] = append(groupByFansub[string(it.Fansub.ID)], it)
	}

	seen := map[string]bool{}
	var groups []model.AnimeGardenGroup
	for _, it := range items {
		fansub := it.Fansub
		if seen[string(fansub.ID)] {
			continue
		}
		seen[string(fansub.ID)] = true
		name := strings.ReplaceAll(fansub.Name, "&", "%26")
		groupItems := groupByFansub[string(fansub.ID)]
		group := model.AnimeGardenGroup{
			ID:    string(fansub.ID),
			Name:  fansub.Name,
			Rss:   fmt.Sprintf("%s/feed.xml?subject=%s&fansub=%s", host, bgmId, name),
			BgmId: bgmId,
		}
		if len(groupItems) > 0 {
			group.LastUpdatedAt = groupItems[0].CreatedAt
		}
		groups = append(groups, group)
	}
	sort.SliceStable(groups, func(i, j int) bool {
		return groups[i].LastUpdatedAt.Time().After(groups[j].LastUpdatedAt.Time())
	})

	for i := range groups {
		g := &groups[i]
		groupItems := groupByFansub[g.ID]
		var titles []string
		for _, it := range groupItems {
			titles = append(titles, it.Title)
		}
		g.Items = groupItems
		g.GroupRegex = groupregex.ToGroupRegex(titles)
	}
	return groups
}