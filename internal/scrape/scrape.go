package scrape

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"ani-rss/internal/config"
	"ani-rss/internal/log"
	"ani-rss/internal/model"
	"ani-rss/internal/service"
	"ani-rss/internal/tmdb"
	"ani-rss/internal/util"
)

var seasonReg = regexp.MustCompile(`[Ss](\d+)[Ee](\d+(\.5)?)`)

// Scrape runs TMDB metadata scraping for an ani into its download path.
func Scrape(ani *model.Ani, force bool) error {
	if ani == nil || ani.Tmdb == nil {
		return fmt.Errorf("缺少 TMDB 信息")
	}
	downloadPath := service.GetDownloadPath(ani)
	isOva := ani.Ova
	log.Infof("scrape", "正在刮削 ... %s", ani.Title)

	if isOva {
		scrapeMovie(ani.Tmdb, downloadPath, force)
	} else {
		scrapeTv(ani.Tmdb, ani.Season, downloadPath, force)
	}
	saveBangumiIni(ani, force)
	log.Infof("scrape", "刮削完成 %s", ani.Title)
	return nil
}

func scrapeMovie(t *model.Tmdb, downloadPath string, force bool) {
	entries, err := os.ReadDir(downloadPath)
	if err != nil {
		return
	}
	var video string
	var maxSize int64
	for _, e := range entries {
		if e.IsDir() || !util.IsVideo(e.Name()) {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			continue
		}
		if fi.Size() > maxSize {
			maxSize = fi.Size()
			video = e.Name()
		}
	}
	if video == "" {
		return
	}
	mainName := strings.TrimSuffix(video, filepath.Ext(video))
	nfoFile := filepath.Join(downloadPath, mainName+".nfo")
	if force || !fileExists(nfoFile) {
		if err := os.WriteFile(nfoFile, []byte(generateMovieNfo(t)), 0o644); err != nil {
			log.Errorf("scrape", "写电影NFO失败 %v", err)
		}
	}
	saveTmdbImages(t, downloadPath, force)
}

func scrapeTv(t *model.Tmdb, season int, downloadPath string, force bool) {
	if !dirExists(downloadPath) {
		return
	}
	parent := filepath.Dir(downloadPath)
	tvShowNfo := filepath.Join(parent, "tvshow.nfo")
	if force || !fileExists(tvShowNfo) {
		if err := os.WriteFile(tvShowNfo, []byte(generateTvShowNfo(t)), 0o644); err != nil {
			log.Errorf("scrape", "写tvshow.nfo失败 %v", err)
		}
	}
	saveTmdbImages(t, parent, force)

	seasonInfo, err := tmdb.GetTvSeason(t, season)
	if err != nil {
		log.Warnf("scrape", "获取季信息失败 %v", err)
		return
	}
	seasonFormat := fmt.Sprintf("%02d", season)

	// season poster in parent dir
	posterPath := seasonInfo.PosterPath
	if posterPath == "" {
		posterPath = t.PosterPath
	}
	if posterPath != "" {
		saveImage(posterPath, filepath.Join(parent, "season"+seasonFormat+"-poster"+extOf(posterPath)), force)
	}

	seasonNfo := filepath.Join(downloadPath, "season.nfo")
	if force || !fileExists(seasonNfo) {
		if err := os.WriteFile(seasonNfo, []byte(generateSeasonNfo(seasonInfo, season)), 0o644); err != nil {
			log.Errorf("scrape", "写season.nfo失败 %v", err)
		}
	}

	episodeMap := map[int]tmdb.TmdbEpisode{}
	for _, e := range seasonInfo.Episodes {
		episodeMap[e.EpisodeNumber] = e
	}

	followDay := config.Get().FollowDay
	followCutoff := time.Now().AddDate(0, 0, -followDay)

	files, err := os.ReadDir(downloadPath)
	if err != nil {
		return
	}
	for _, f := range files {
		if f.IsDir() || !util.IsVideo(f.Name()) {
			continue
		}
		mainName := strings.TrimSuffix(f.Name(), filepath.Ext(f.Name()))
		m := seasonReg.FindStringSubmatch(mainName)
		if len(m) < 3 {
			continue
		}
		seasonNumber := atoi(m[1])
		if seasonNumber != season {
			continue
		}
		episodeNumber := atoi(m[2])
		ep, ok := episodeMap[episodeNumber]
		if !ok {
			continue
		}
		airDate := parseDate(ep.AirDate)
		isFollow := airDate.IsZero() || !airDate.Before(followCutoff)

		if ep.StillPath != "" {
			thumbFile := filepath.Join(downloadPath, mainName+"-thumb"+extOf(ep.StillPath))
			saveImage(ep.StillPath, thumbFile, isFollow || force)
		}
		epNfo := filepath.Join(downloadPath, mainName+".nfo")
		if isFollow || force || !fileExists(epNfo) {
			if err := os.WriteFile(epNfo, []byte(generateEpisodeNfo(ep)), 0o644); err != nil {
				log.Errorf("scrape", "写剧集NFO失败 %v", err)
			}
		}
	}
}

func saveTmdbImages(t *model.Tmdb, outputPath string, force bool) {
	poster := t.PosterPath
	fanart := t.BackdropPath
	if poster != "" {
		saveImage(poster, filepath.Join(outputPath, "poster"+extOf(poster)), force)
	}
	if fanart != "" {
		saveImage(fanart, filepath.Join(outputPath, "fanart"+extOf(fanart)), force)
	}
	imgs, err := tmdb.GetTmdbImages(t)
	if err != nil {
		return
	}
	// backdrops != main fanart, width >= 1280, max 4
	var extra int
	for _, b := range imgs.Backdrops {
		if b.FilePath == fanart {
			continue
		}
		if b.Width < 1280 {
			continue
		}
		if extra >= 4 {
			break
		}
		extra++
		saveImage(b.FilePath, filepath.Join(outputPath, fmt.Sprintf("fanart%d%s", extra, extOf(b.FilePath))), force)
	}
	if len(imgs.Logos) > 0 {
		saveImage(imgs.Logos[0].FilePath, filepath.Join(outputPath, "clearlogo"+extOf(imgs.Logos[0].FilePath)), force)
	}
}

func saveImage(tmdbPath, saveFile string, force bool) {
	if tmdbPath == "" {
		return
	}
	if !force && fileExists(saveFile) {
		return
	}
	_ = os.Remove(saveFile)
	b, err := util.GetBytes(tmdb.ImageURL(tmdbPath))
	if err != nil {
		log.Warnf("scrape", "下载图片失败 %s %v", tmdbPath, err)
		return
	}
	if err := os.MkdirAll(filepath.Dir(saveFile), 0o755); err != nil {
		return
	}
	if err := os.WriteFile(saveFile, b, 0o644); err != nil {
		return
	}
	log.Infof("scrape", "已保存图片 %s", saveFile)
}

func saveBangumiIni(ani *model.Ani, force bool) {
	if !config.Get().BangumiIniEnabled {
		return
	}
	downloadPath := service.GetDownloadPath(ani)
	file := filepath.Join(downloadPath, "bangumi.ini")
	if !force && fileExists(file) {
		return
	}
	subjectId := ""
	if renameHook != nil {
		subjectId = renameHook(ani)
	}
	content := fmt.Sprintf("[Bangumi]\nid=%s\noffset=%d\n", subjectId, ani.Offset)
	_ = os.WriteFile(file, []byte(content), 0o644)
	log.Infof("scrape", "已保存 %s", file)
}

var renameHook func(ani *model.Ani) string

// SetBgmSubjectIdHook wires the bgm subject id resolver.
func SetBgmSubjectIdHook(fn func(ani *model.Ani) string) { renameHook = fn }

func xmlEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;", "'", "&apos;")
	return r.Replace(s)
}

func generateMovieNfo(t *model.Tmdb) string {
	var sb strings.Builder
	sb.WriteString("<?xml version=\"1.0\" encoding=\"utf-8\" standalone=\"yes\"?>\n<movie>\n")
	writeNfoFields(&sb, t)
	sb.WriteString("</movie>\n")
	return sb.String()
}

func generateTvShowNfo(t *model.Tmdb) string {
	var sb strings.Builder
	sb.WriteString("<?xml version=\"1.0\" encoding=\"utf-8\" standalone=\"yes\"?>\n<tvshow>\n")
	writeNfoFields(&sb, t)
	if t.TmdbGroupId != "" {
		sb.WriteString("    <tmdbegid>" + xmlEscape(t.TmdbGroupId) + "</tmdbegid>\n")
	}
	for _, g := range t.Genres {
		sb.WriteString("    <genre>" + xmlEscape(g.Name) + "</genre>\n")
	}
	for _, c := range t.Cast {
		sb.WriteString("    <actor>\n        <name>" + xmlEscape(c.Name) + "</name>\n        <role>" + xmlEscape(c.Character) + "</role>\n        <type>Actor</type>\n        <tmdbid>" + fmt.Sprint(c.ID) + "</tmdbid>\n    </actor>\n")
	}
	for _, v := range t.Videos {
		if strings.EqualFold(v.Site, "YouTube") && v.Key != "" {
			sb.WriteString("    <trailer>https://www.youtube.com/watch?v=" + xmlEscape(v.Key) + "</trailer>\n")
		}
	}
	for _, n := range t.Networks {
		sb.WriteString("    <studio>" + xmlEscape(n.Name) + "</studio>\n")
	}
	sb.WriteString("</tvshow>\n")
	return sb.String()
}

func writeNfoFields(sb *strings.Builder, t *model.Tmdb) {
	sb.WriteString("    <tmdbid>" + fmt.Sprint(t.ID) + "</tmdbid>\n")
	sb.WriteString("    <title>" + xmlEscape(t.Name) + "</title>\n")
	if t.OriginalName != "" {
		sb.WriteString("    <originaltitle>" + xmlEscape(t.OriginalName) + "</originaltitle>\n")
	}
	if !t.Date.Time().IsZero() {
		sb.WriteString("    <year>" + fmt.Sprint(t.Date.Time().Year()) + "</year>\n")
		sb.WriteString("    <releasedate>" + t.Date.Time().Format("2006-01-02") + "</releasedate>\n")
	}
	if t.Overview != "" {
		sb.WriteString("    <plot>" + xmlEscape(t.Overview) + "</plot>\n")
	}
	if t.VoteAverage != 0 {
		sb.WriteString("    <rating>" + fmt.Sprintf("%.1f", t.VoteAverage) + "</rating>\n")
	}
	if t.VoteCount != 0 {
		sb.WriteString("    <votes>" + fmt.Sprint(t.VoteCount) + "</votes>\n")
	}
	if t.Tagline != "" {
		sb.WriteString("    <tagline>" + xmlEscape(t.Tagline) + "</tagline>\n")
	}
	if t.Runtime != 0 {
		sb.WriteString("    <runtime>" + fmt.Sprint(t.Runtime) + "</runtime>\n")
	}
}

func generateSeasonNfo(s *tmdb.TmdbSeason, season int) string {
	var sb strings.Builder
	sb.WriteString("<?xml version=\"1.0\" encoding=\"utf-8\" standalone=\"yes\"?>\n<season>\n")
	if s.Name != "" {
		sb.WriteString("    <title>" + xmlEscape(s.Name) + "</title>\n")
	}
	if s.Overview != "" {
		sb.WriteString("    <plot>" + xmlEscape(s.Overview) + "</plot>\n")
		sb.WriteString("    <outline>" + xmlEscape(s.Overview) + "</outline>\n")
	}
	sb.WriteString("    <seasonnumber>" + fmt.Sprint(season) + "</seasonnumber>\n")
	if d := parseDate(s.AirDate); !d.IsZero() {
		sb.WriteString("    <year>" + fmt.Sprint(d.Year()) + "</year>\n")
		sb.WriteString("    <releasedate>" + d.Format("2006-01-02") + "</releasedate>\n")
	}
	sb.WriteString("</season>\n")
	return sb.String()
}

func generateEpisodeNfo(e tmdb.TmdbEpisode) string {
	var sb strings.Builder
	sb.WriteString("<?xml version=\"1.0\" encoding=\"utf-8\" standalone=\"yes\"?>\n<episodedetails>\n")
	if e.Name != "" {
		sb.WriteString("    <title>" + xmlEscape(e.Name) + "</title>\n")
	}
	if e.Overview != "" {
		sb.WriteString("    <plot>" + xmlEscape(e.Overview) + "</plot>\n")
	}
	if e.VoteAverage != 0 {
		sb.WriteString("    <rating>" + fmt.Sprintf("%.1f", e.VoteAverage) + "</rating>\n")
	}
	if d := parseDate(e.AirDate); !d.IsZero() {
		sb.WriteString("    <year>" + fmt.Sprint(d.Year()) + "</year>\n")
		sb.WriteString("    <aired>" + d.Format("2006-01-02") + "</aired>\n")
	}
	sb.WriteString("    <episode>" + fmt.Sprint(e.EpisodeNumber) + "</episode>\n")
	season := e.SeasonNumber
	if season <= 0 {
		season = 0
	}
	sb.WriteString("    <season>" + fmt.Sprint(season) + "</season>\n")
	if e.Runtime != 0 {
		sb.WriteString("    <runtime>" + fmt.Sprint(e.Runtime) + "</runtime>\n")
	}
	sb.WriteString("</episodedetails>\n")
	return sb.String()
}

func extOf(p string) string {
	ext := strings.ToLower(filepath.Ext(p))
	if ext == ".jpeg" {
		return ".jpg"
	}
	return ext
}

func parseDate(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	if t, err := time.ParseInLocation("2006-01-02", s, model.Loc()); err == nil {
		return t
	}
	return time.Time{}
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func dirExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return n
		}
		n = n*10 + int(c-'0')
	}
	return n
}