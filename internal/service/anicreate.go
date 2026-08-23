package service

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"ani-rss/internal/bgm"
	"ani-rss/internal/config"
	"ani-rss/internal/mikan"
	"ani-rss/internal/model"
	"ani-rss/internal/rename"
	"ani-rss/internal/rss"
	"ani-rss/internal/util"
)

// SaveCover downloads an image url to files/ and returns the relative path.
func SaveCover(imageURL string) string {
	if imageURL == "" {
		return ""
	}
	b, err := util.GetBytes(imageURL)
	if err != nil {
		return ""
	}
	hash := util.MD5Hex(imageURL)
	ext := coverExt(imageURL)
	if ext == "" {
		ext = ".jpg"
	}
	rel := filepath.ToSlash(filepath.Join(hash[:1], hash+ext))
	full := config.ConfigDirFile("files/" + rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return ""
	}
	if err := os.WriteFile(full, b, 0o644); err != nil {
		return ""
	}
	return rel
}

func coverExt(rawURL string) string {
	p := rawURL
	if i := strings.Index(p, "?"); i >= 0 {
		p = p[:i]
	}
	switch strings.ToLower(filepath.Ext(p)) {
	case ".png", ".jpg", ".jpeg", ".webp", ".gif":
		if strings.HasSuffix(strings.ToLower(p), ".jpeg") {
			return ".jpg"
		}
		return strings.ToLower(filepath.Ext(p))
	}
	return ""
}

func parseQuery(rawURL string) map[string]string {
	out := map[string]string{}
	if u, err := url.Parse(rawURL); err == nil {
		for k, v := range u.Query() {
			if len(v) > 0 {
				out[k] = v[0]
			}
		}
	}
	return out
}

// RssToAni builds an Ani from an RSS url + bgmUrl, filling animation info
// from BGM (mirrors AniUtil.getAni + BgmUtil.toAni).
func RssToAni(dto *model.RssToAniDTO) (*model.Ani, error) {
	urlStr := strings.TrimSpace(dto.URL)
	if urlStr == "" {
		return nil, errors.New("RSS地址 不能为空")
	}
	typ := dto.Type
	if typ == "" {
		typ = "mikan"
	}
	enable := dto.Enable
	if !dto.Enable {
		enable = false
	} else {
		enable = true
	}

	ani := model.DefaultAni()
	ani.URL = urlStr
	params := parseQuery(urlStr)

	switch typ {
	case "mikan":
		subgroup := dto.Subgroup
		bgmUrl := dto.BgmUrl
		if strings.TrimSpace(subgroup) == "" && strings.TrimSpace(bgmUrl) == "" {
			subgroupId := mikan.GetSubgroupId(urlStr)
			// fill from mikan page
			if mikanId := mikanBangumiId(urlStr); mikanId != "" {
				if info := mikan.GetMikanInfo(mikanId); info != nil {
					if info.BgmUrl != "" {
						bgmUrl = info.BgmUrl
					}
					if subgroupId != "" {
						for _, g := range info.Groups {
							if g.SubgroupId == subgroupId {
								subgroup = g.Label
								break
							}
						}
					}
				}
			}
		}
		ani.BgmUrl = bgmUrl
		ani.Subgroup = subgroup
	case "ani-bt":
		if bgmId := params["bgmId"]; bgmId != "" {
			ani.BgmUrl = "https://bgm.tv/subject/" + bgmId
		}
		subgroup := dto.Subgroup
		if slug := params["groupSlug"]; slug != "" && subgroup == "" {
			subgroup = slug
		}
		ani.Subgroup = subgroup
	case "anime-garden":
		if subj := params["subject"]; subj != "" {
			ani.BgmUrl = "https://bgm.tv/subject/" + subj
		}
		if fansub := params["fansub"]; fansub != "" {
			ani.Subgroup = fansub
		}
	default:
		ani.BgmUrl = dto.BgmUrl
	}

	if strings.TrimSpace(ani.BgmUrl) == "" {
		return nil, errors.New("bgmUrl 不能为空")
	}

	subjectId := bgm.GetSubjectIdByUrl(ani.BgmUrl)
	if subjectId == "" {
		return nil, errors.New("无法解析 BGM 番剧ID")
	}
	info, err := bgm.GetBgmInfo(subjectId)
	if err != nil {
		return nil, errors.New("获取 BGM 信息失败: " + err.Error())
	}
	bgm.ToAni(info, ani)

	cfg := config.Get()
	ani.DownloadNew = cfg.DownloadNew
	ani.GlobalExclude = cfg.EnabledExclude
	if cfg.ImportExclude {
		merged := append([]string{}, cfg.Exclude...)
		merged = append(merged, ani.Exclude...)
		ani.Exclude = dedupStrs(merged)
	}
	ani.Type = typ
	ani.Enable = enable

	subgroup := ani.Subgroup
	if strings.TrimSpace(subgroup) == "" {
		subgroup = "未知字幕组"
	}
	if subgroup == "未知字幕组" {
		if items := rss.GetItems(ani); len(items) > 0 {
			subgroup = rename.GetSubgroup(items)
		}
	}
	ani.Subgroup = subgroup

	// copyMasterToStandby
	if cfg.CopyMasterToStandby && cfg.StandbyRss {
		ani.StandbyRssList = append(ani.StandbyRssList, model.StandbyRss{
			URL:    urlStr,
			Offset: 0,
			Label:  subgroup,
		})
	}

	// auto infer episode offset
	if cfg.Offset {
		if items := rss.GetItems(ani); len(items) > 0 {
			minEp := items[0].Episode
			for _, it := range items {
				if it.Episode < minEp {
					minEp = it.Episode
				}
			}
			offset := -(int(minEp) - 1)
			ani.Offset = offset
			for i := range ani.StandbyRssList {
				ani.StandbyRssList[i].Offset = offset
			}
		}
	}

	// download path templates
	ani.CustomDownloadPathTemplate = GetDownloadPath(ani)
	ani.CustomCompletedPathTemplate = cfg.CompletedPathTemplate
	return ani, nil
}

func mikanBangumiId(rawURL string) string {
	if i := strings.LastIndex(rawURL, "/Home/Bangumi/"); i >= 0 {
		id := rawURL[i+len("/Home/Bangumi/"):]
		id = strings.TrimRight(id, "/")
		if j := strings.IndexAny(id, "?#"); j >= 0 {
			id = id[:j]
		}
		if id != "" {
			return id
		}
	}
	return ""
}

func dedupStrs(list []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range list {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}