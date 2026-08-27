package groupregex

import (
	"encoding/json"
	"regexp"

	"ani-rss/internal/model"
)

var regexList = []string{
	"1920[Xx]1080", "3840[Xx]2160",
	"1080[Pp]", "720[Pp]", "4[Kk]",
	"繁", "简", "日",
	"内嵌", "内封", "外挂",
	"cht|Cht|CHT", "chs|Chs|CHS",
	"avc|Avc|AVC", "hevc|Hevc|HEVC",
	"h264|H264", "h265|H265",
	"10bit|10Bit|10BIT",
	"mp4|MP4", "mkv|MKV",
}

var compiledRegexes = func() []*regexp.Regexp {
	out := make([]*regexp.Regexp, 0, len(regexList))
	for _, r := range regexList {
		re, err := regexp.Compile(r)
		if err != nil {
			continue
		}
		out = append(out, re)
	}
	return out
}()

func dedupStrings(list []string) []string {
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

func dedupRegexItems(list []model.GroupRegexItem) []model.GroupRegexItem {
	seen := map[string]bool{}
	var out []model.GroupRegexItem
	for _, it := range list {
		b, _ := json.Marshal(it)
		if seen[string(b)] {
			continue
		}
		seen[string(b)] = true
		out = append(out, it)
	}
	return out
}

func dedupRegexList(list [][]model.GroupRegexItem) [][]model.GroupRegexItem {
	seen := map[string]bool{}
	var out [][]model.GroupRegexItem
	for _, it := range list {
		b, _ := json.Marshal(it)
		if seen[string(b)] {
			continue
		}
		seen[string(b)] = true
		out = append(out, it)
	}
	return out
}

// ToGroupRegex builds fansub filter rules from a list of release titles,
// mirroring GroupRegexUtils.toGroupRegx.
func ToGroupRegex(titles []string) model.GroupRegex {
	titles = dedupStrings(titles)

	var regexListOut [][]model.GroupRegexItem
	var tags []string

	for _, title := range titles {
		var regexItems []model.GroupRegexItem
		for _, re := range compiledRegexes {
			if !re.MatchString(title) {
				continue
			}
			tag := re.FindString(title)
			regexItems = append(regexItems, model.GroupRegexItem{
				Regex: re.String(),
				Label: tag,
			})
			if len(tags) < 5 && !containsStr(tags, tag) {
				tags = append(tags, tag)
			}
		}
		if len(regexItems) == 0 {
			continue
		}
		regexItems = dedupRegexItems(regexItems)
		regexListOut = append(regexListOut, regexItems)
	}

	regexListOut = dedupRegexList(regexListOut)

	return model.GroupRegex{
		RegexList: regexListOut,
		Tags:      tags,
	}
}

func containsStr(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}