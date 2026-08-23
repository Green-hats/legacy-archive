package rename

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"ani-rss/internal/config"
	"ani-rss/internal/model"
)

// RegStr is the episode extraction regex (RenameUtil.REG_STR).
var RegStr = `(.*|\[.*])(( - |Vol |[Ee][Pp]?)\d+(\.5)?( ?\(\d+\))?|【\d+(\.5)?】|\[\d+(\.5)?( ?\(\d+\))?( ?[vV]\d)?( ?END)?( ?完)?( ?FIN)?]|第\d+(\.5)?[话話集]( - END)?|^\[TOC].* \d+|^六四位元字幕组.*★\d+(\.5)?★)`

var (
	regEp       = regexp.MustCompile(RegStr)
	regNumber   = regexp.MustCompile(`\d+(\.5)?`)
	regYear     = regexp.MustCompile(` ?\(((19|20)\d{2})\)`)
	regTmdbId   = regexp.MustCompile(` ?(\[tmdbid=(\d+)]|\{tmdb-(\d+)})`)
	regRes      = regexp.MustCompile(`(720|1080|2160)[Pp]`)
	regHashTail = regexp.MustCompile(`\[([A-Z]|\d){8}]$`)
)

// Rename extracts the episode and builds the reName for an item.
// Returns false if no episode could be parsed (item should be dropped).
func Rename(ani *model.Ani, item *model.Item) bool {
	offset := ani.Offset
	season := ani.Season
	title := ani.Title

	if ani.Ova {
		item.ReName = RenameDel(title)
		return true
	}

	itemTitle := item.Title
	itemTitle = strings.ReplaceAll(itemTitle, "+NCOPED", "")
	itemTitle = strings.TrimSpace(itemTitle)
	itemTitle = strings.ReplaceAll(itemTitle, "\n", " ")
	itemTitle = strings.ReplaceAll(itemTitle, "\t", " ")
	itemTitle = regHashTail.ReplaceAllString(itemTitle, "")
	itemTitle = strings.TrimSpace(itemTitle)

	var e string
	if ani.CustomEpisode {
		re, err := regexp.Compile(ani.CustomEpisodeStr)
		if err == nil {
			groups := re.FindStringSubmatch(itemTitle)
			idx := ani.CustomEpisodeGroupIndex
			if idx < len(groups) {
				e = groups[idx]
			}
		}
	} else {
		groups := regEp.FindStringSubmatch(itemTitle)
		if len(groups) > 2 {
			e = groups[2]
		}
	}

	if strings.TrimSpace(e) == "" {
		return false
	}

	episodeStr := regNumber.FindString(e)
	if episodeStr == "" {
		return false
	}

	episodeF, err := strconv.ParseFloat(episodeStr, 64)
	if err != nil {
		return false
	}
	episode := episodeF + float64(offset)
	item.Episode = episode

	seasonFormat := fmt.Sprintf("%02d", season)
	episodeFormat := fmt.Sprintf("%02d", int(episode))
	episodeStrInt := strconv.Itoa(int(episode))

	is5 := isHalf(episode)

	if config.Get().Skip5 && is5 {
		return false
	}

	if is5 {
		episodeFormat = episodeFormat + ".5"
		episodeStrInt = episodeStrInt + ".5"
	}

	itemTitle = GetName(itemTitle)
	resolution := GetResolution(itemTitle)
	tmdbId := ""
	if ani.Tmdb != nil && ani.Tmdb.ID != 0 {
		tmdbId = strconv.Itoa(ani.Tmdb.ID)
	}

	subgroup := item.Subgroup
	if strings.TrimSpace(subgroup) == "" {
		subgroup = "未知字幕组"
	}

	tmpl := GetRenameTemplate(ani)
	tmpl = strings.ReplaceAll(tmpl, "${seasonFormat}", seasonFormat)
	tmpl = strings.ReplaceAll(tmpl, "${episodeFormat}", episodeFormat)
	tmpl = strings.ReplaceAll(tmpl, "${season}", strconv.Itoa(season))
	tmpl = strings.ReplaceAll(tmpl, "${episode}", episodeStrInt)
	tmpl = strings.ReplaceAll(tmpl, "${subgroup}", subgroup)
	tmpl = strings.ReplaceAll(tmpl, "${itemTitle}", itemTitle)
	tmpl = strings.ReplaceAll(tmpl, "${resolution}", resolution)
	tmpl = strings.ReplaceAll(tmpl, "${tmdbid}", tmdbId)
	tmpl = strings.ReplaceAll(tmpl, "${title}", title)
	tmpl = ReplaceEpisodeTitle(tmpl, episode, ani)
	if strings.Contains(tmpl, "${bgmId}") {
		tmpl = strings.ReplaceAll(tmpl, "${bgmId}", BgmSubjectId(ani))
	}
	if strings.Contains(tmpl, "${jpTitle}") {
		tmpl = strings.ReplaceAll(tmpl, "${jpTitle}", JpTitle(ani))
	}
	tmpl = strings.ReplaceAll(tmpl, "${themoviedbName}", ani.ThemoviedbName)
	tmpl = RenameDel(tmpl)

	reName := GetName(tmpl)
	if maxLen := config.Get().MaxFileNameLength; maxLen > 0 && len([]rune(reName)) > maxLen {
		reName = string([]rune(reName)[:maxLen])
	}
	item.ReName = reName
	return true
}

// GetRenameTemplate resolves the effective rename template for an ani.
func GetRenameTemplate(ani *model.Ani) string {
	tmpl := config.Get().RenameTemplate
	if ani.CustomRenameTemplateEnable {
		tmpl = ani.CustomRenameTemplate
	}
	if strings.TrimSpace(tmpl) == "" {
		tmpl = "${title} S${seasonFormat}E${episodeFormat}"
	}
	return tmpl
}

func isHalf(ep float64) bool {
	return int(ep) != int(ep+0.499999) && ep != float64(int(ep))
}

// Is5 reports whether an episode is an x.5 special.
func Is5(ep float64) bool {
	return ep != float64(int(ep))
}

// GetResolution extracts the video resolution from a title.
func GetResolution(itemTitle string) string {
	repl := map[string]string{
		"1920x1080": "1080p",
		"3840x2160": "2160p",
		"1280x720":  "720p",
	}
	for k, v := range repl {
		itemTitle = strings.ReplaceAll(itemTitle, k, v)
	}
	if regRes.MatchString(itemTitle) {
		return strings.ToLower(regRes.FindString(itemTitle))
	}
	return "none"
}

// GetName sanitizes a filename (getName).
func GetName(s string) string {
	if strings.TrimSpace(s) == "" {
		return ""
	}
	s = strings.ReplaceAll(s, "1/2", "½")
	repl := map[string]string{
		"/": " ",
		"\\": " ",
		":": "：",
		"?": "？",
		"|": "｜",
		"*": " ",
		"<": " ",
		">": " ",
		"\"": " ",
	}
	for k, v := range repl {
		s = strings.ReplaceAll(s, k, v)
	}
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return strings.TrimSpace(s)
}

// RenameDel strips tmdb id and year markers according to config.
func RenameDel(title string) string {
	return RenameDelConfig(title, true)
}

// RenameDelConfig strips tmdb id / year markers, honoring or ignoring config.
func RenameDelConfig(title string, isConfig bool) string {
	if strings.TrimSpace(title) == "" {
		return ""
	}
	if !isConfig {
		title = regTmdbId.ReplaceAllString(title, "")
		title = regYear.ReplaceAllString(title, "")
		return strings.TrimSpace(title)
	}
	cfg := config.Get()
	if cfg.RenameDelTmdbId {
		title = regTmdbId.ReplaceAllString(title, "")
	}
	if cfg.RenameDelYear {
		title = regYear.ReplaceAllString(title, "")
	}
	return strings.TrimSpace(title)
}

// ReplaceEpisodeTitle substitutes episode title tokens using TMDB/BGM data.
func ReplaceEpisodeTitle(template string, episode float64, ani *model.Ani) string {
	needTmdb := strings.Contains(template, "${episodeTitle}")
	needBgm := strings.Contains(template, "${bgmEpisodeTitle}") || strings.Contains(template, "${bgmJpEpisodeTitle}")

	defaultTitle := fmt.Sprintf("第%s集", formatNum(episode))
	episodeTitle, bgmEpisodeTitle, bgmJpEpisodeTitle := defaultTitle, defaultTitle, defaultTitle

	if !Is5(episode) {
		epInt := int(episode)
		if needTmdb {
			if name, ok := TmdbEpisodeTitle(ani, epInt); ok {
				episodeTitle = name
			}
		}
		if needBgm {
			if cn, jp, ok := BgmEpisodeTitle(ani, epInt); ok {
				bgmEpisodeTitle = cn
				bgmJpEpisodeTitle = jp
			}
		}
	}
	template = strings.ReplaceAll(template, "${episodeTitle}", episodeTitle)
	template = strings.ReplaceAll(template, "${bgmEpisodeTitle}", bgmEpisodeTitle)
	template = strings.ReplaceAll(template, "${bgmJpEpisodeTitle}", bgmJpEpisodeTitle)
	return template
}

func formatNum(ep float64) string {
	if Is5(ep) {
		return fmt.Sprintf("%d.5", int(ep))
	}
	return strconv.Itoa(int(ep))
}

// GetSubgroup extracts the fansub group from the first bracketed item title.
func GetSubgroup(items []*model.Item) string {
	reg := regexp.MustCompile(`^\[(.+?)]`)
	for _, item := range items {
		title := item.Title
		if m := reg.FindStringSubmatch(title); len(m) > 1 {
			return m[1]
		}
	}
	return "未知字幕组"
}

// The following hooks are registered by the bgm/tmdb packages to avoid
// import cycles.
var (
	// BgmSubjectId resolves the bangumi subject id for an ani.
	BgmSubjectId func(ani *model.Ani) string
	// JpTitle resolves the Japanese title for an ani.
	JpTitle func(ani *model.Ani) string
	// TmdbEpisodeTitle resolves a TMDB episode name (ani, ep) -> name.
	TmdbEpisodeTitle func(ani *model.Ani, ep int) (string, bool)
	// BgmEpisodeTitle resolves BGM episode names (ani, ep) -> (nameCn, name).
	BgmEpisodeTitle func(ani *model.Ani, ep int) (string, string, bool)
)