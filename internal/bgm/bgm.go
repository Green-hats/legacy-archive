package bgm

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"ani-rss/internal/cache"
	"ani-rss/internal/config"
	"ani-rss/internal/model"
	"ani-rss/internal/util"
)

var (
	subjectIdRe = regexp.MustCompile(`/subject/(\d+)`)
)

// SetToken adds the BGM bearer token header.
func SetToken(req *http.Request) *http.Request {
	if token := config.Get().BgmToken; token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("User-Agent", "ani-rss/"+util.Version+" (https://github.com/wushuo894/ani-rss)")
	req.Header.Set("Accept", "application/json")
	return req
}

func bgmGet(path string) (*http.Response, error) {
	req, err := http.NewRequest("GET", config.Get().BgmApi+path, nil)
	if err != nil {
		return nil, err
	}
	SetToken(req)
	return util.DefaultClient().Do(req)
}

func bgmPostForm(rawURL string, form url.Values) (*http.Response, error) {
	req, err := http.NewRequest("POST", rawURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return util.DefaultClient().Do(req)
}

func bgmPostJSON(path string, body interface{}) (*http.Response, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", config.Get().BgmApi+path, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	SetToken(req)
	return util.DefaultClient().Do(req)
}

// Search searches bangumi subjects by name.
func Search(name string) []*model.BgmInfo {
	if strings.TrimSpace(name) == "" {
		return nil
	}
	name = strings.ReplaceAll(name, "1/2", "½")
	u := config.Get().BgmApi + "/search/subject/" + url.PathEscape(name) + "?type=2&max_results=25&responseGroup=small"
	resp, err := bgmGet(u)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil
	}
	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil
	}
	list, _ := body["list"].([]interface{})
	var out []*model.BgmInfo
	for _, item := range list {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		info := &model.BgmInfo{
			ID:       strval(m["id"]),
			URL:      "https://bgm.tv/subject/" + strval(m["id"]),
			Name:     strval(m["name"]),
			NameCn:   strval(m["name_cn"]),
			Eps:      intval(m["eps"]),
			Platform: strval(m["type"]),
			Season:   intval(m["air_date"] == nil),
		}
		if dateStr, ok := m["air_date"].(string); ok && dateStr != "" {
			if t, err := time.ParseInLocation("2006-01-02", dateStr, model.Loc()); err == nil {
				info.Date = model.DateTime(t)
			}
		}
		if images, ok := m["images"].(map[string]interface{}); ok {
			info.Images = model.BgmImages{
				Small:  strval(images["small"]),
				Grid:   strval(images["grid"]),
				Large:  strval(images["large"]),
				Medium: strval(images["medium"]),
				Common: strval(images["common"]),
			}
		}
		if rating, ok := m["rating"].(map[string]interface{}); ok {
			info.Rating.Score = floatval(rating["score"])
			info.Rating.Total = intval(rating["total"])
		}
		if tags, ok := m["tags"].([]interface{}); ok {
			for _, t := range tags {
				tm, ok := t.(map[string]interface{})
				if !ok {
					continue
				}
				info.Tags = append(info.Tags, model.BgmTag{
					Name:      strval(tm["name"]),
					Count:     intval(tm["count"]),
					TotalCont: intval(tm["totalCont"]),
				})
			}
		}
		out = append(out, info)
	}
	return out
}

// GetSubjectIdByUrl extracts the subject id from a bgm url.
func GetSubjectIdByUrl(bgmUrl string) string {
	if m := subjectIdRe.FindStringSubmatch(bgmUrl); len(m) > 1 {
		return m[1]
	}
	return ""
}

// GetSubjectIdByName resolves the subject id for a title (cached).
func GetSubjectIdByName(name string, season int) string {
	if strings.TrimSpace(name) == "" {
		return ""
	}
	key := fmt.Sprintf("BGM_getSubjectId:%s", name)
	if v, ok := cache.Default.Get(key); ok {
		return v.(string)
	}
	list := Search(name)
	var id string
	for _, info := range list {
		if season > 0 && info.Season > 0 && info.Season != season {
			continue
		}
		if info.Name == name || info.NameCn == name {
			id = info.ID
			break
		}
	}
	if id == "" && len(list) > 0 {
		id = list[0].ID
	}
	cache.Default.PutDuration(key, id, 10*time.Minute)
	return id
}

// GetFinalName returns the display title for a bgm subject (mirrors BgmUtil.getFinalName).
func GetFinalName(info *model.BgmInfo) string {
	if info == nil {
		return "无标题"
	}
	title := info.NameCn
	if title == "" {
		title = info.Name
	}
	if config.Get().BgmJpName {
		title = info.Name
	}
	if strings.TrimSpace(title) == "" {
		title = "无标题"
	}
	return strings.TrimSpace(title)
}

var seasonRegexes = []*regexp.Regexp{
	regexp.MustCompile(`第 ?([一二三四五六七八九十百千]+) ?[季期]`),
	regexp.MustCompile(`[Ss]eason ?(\d+)`),
	regexp.MustCompile(`(\d+)(st|nd|rd|th) ?[Ss]eason`),
	regexp.MustCompile(`[Ss](\d+)$`),
}

func chineseNum(s string) int {
	digits := map[rune]int{'一': 1, '二': 2, '三': 3, '四': 4, '五': 5, '六': 6, '七': 7, '八': 8, '九': 9}
	// handle 十/百/千 simply
	if s == "十" {
		return 10
	}
	total := 0
	for _, r := range s {
		if v, ok := digits[r]; ok {
			total = total*10 + v
		}
	}
	// "十五" -> 15
	if strings.Contains(s, "十") {
		parts := strings.Split(s, "十")
		left := 0
		if parts[0] != "" {
			left = chineseDigits(parts[0])
		}
		right := 0
		if len(parts) > 1 && parts[1] != "" {
			right = chineseDigits(parts[1])
		}
		val := left*10 + right
		if val > 0 {
			return val
		}
	}
	return total
}

func chineseDigits(s string) int {
	digits := map[rune]int{'一': 1, '二': 2, '三': 3, '四': 4, '五': 5, '六': 6, '七': 7, '八': 8, '九': 9}
	n := 0
	for _, r := range s {
		if v, ok := digits[r]; ok {
			n = n*10 + v
		}
	}
	return n
}

// GetSeasonByName infers the season number from a title (mirrors BgmUtil.getSeasonByName).
func GetSeasonByName(name string) int {
	if name == "" {
		return 1
	}
	for _, re := range seasonRegexes {
		m := re.FindStringSubmatch(name)
		if len(m) < 2 {
			continue
		}
		s := m[1]
		if n, err := strconv.Atoi(s); err == nil {
			return n
		}
		if n := chineseNum(s); n > 0 {
			return n
		}
	}
	return 1
}

// GetSeason infers the season from tags/titles/alias (mirrors BgmUtil.getSeasonByBgmInfo).
func GetSeason(info *model.BgmInfo) int {
	if info == nil {
		return 1
	}
	for _, tag := range info.Tags {
		if n := GetSeasonByName(tag.Name); n > 1 {
			return n
		}
	}
	if info.NameCn != "" {
		if n := GetSeasonByName(info.NameCn); n > 1 {
			return n
		}
	}
	if info.Name != "" {
		if n := GetSeasonByName(info.Name); n > 1 {
			return n
		}
	}
	return 1
}

// GetEps returns the episode count for a subject.
func GetEps(info *model.BgmInfo) int {
	if info == nil || info.Eps < 1 {
		return 0
	}
	eps := GetEpisodes(info.ID, 0)
	if len(eps) > 0 {
		return len(eps)
	}
	return info.Eps
}

// SaveCoverFn downloads a cover image and returns the local relative path
// (wired by the service layer to avoid an import cycle).
var SaveCoverFn func(imageURL string) string

// ToAni fills animation info from bgm subject into an Ani
// (mirrors BgmUtil.toAni).
func ToAni(info *model.BgmInfo, ani *model.Ani) *model.Ani {
	if info == nil || ani == nil {
		return ani
	}
	cfg := config.Get()
	image := ""
	if cfg.BgmImage != "" {
		image = imageField(&info.Images, cfg.BgmImage)
	} else {
		image = info.Images.Large
	}
	score := 0.0
	if info.Rating.Score != 0 {
		score = info.Rating.Score
	}
	platform := strings.ToUpper(info.Platform)
	ova := platform == "OVA" || platform == "剧场版"

	ani.BgmUrl = "https://bgm.tv/subject/" + info.ID
	ani.Title = GetFinalName(info)
	ani.JpTitle = info.Name
	ani.Season = GetSeason(info)
	ani.TotalEpisodeNumber = GetEps(info)
	ani.Ova = ova
	ani.Score = score
	if !info.Date.Time().IsZero() {
		ani.ReleaseDate = model.Date(info.Date.Time())
	}
	ani.Image = image
	if SaveCoverFn != nil && image != "" {
		ani.Cover = SaveCoverFn(image)
	}
	// completed path template / download path are filled by the service layer
	return ani
}

func imageField(images *model.BgmImages, key string) string {
	switch key {
	case "small":
		return images.Small
	case "medium":
		return images.Medium
	case "grid":
		return images.Grid
	case "common":
		return images.Common
	default:
		return images.Large
	}
}

// GetSubjectId resolves the subject id from an ani (bgmUrl or title).
func GetSubjectId(ani *model.Ani) string {
	if ani == nil {
		return ""
	}
	if id := GetSubjectIdByUrl(ani.BgmUrl); id != "" {
		return id
	}
	return GetSubjectIdByName(ani.Title, ani.Season)
}

// GetBgmInfo fetches subject info by subject id (cached 10 min).
func GetBgmInfo(subjectId string) (*model.BgmInfo, error) {
	if subjectId == "" {
		return nil, errors.New("subjectId is blank")
	}
	key := "BGM_info:" + subjectId
	if v, ok := cache.Default.Get(key); ok {
		if info, ok := v.(*model.BgmInfo); ok {
			return info, nil
		}
	}
	resp, err := bgmGet("/v0/subjects/" + subjectId)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.New("bgm http " + resp.Status)
	}
	var m map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return nil, err
	}
	info := parseSubject(m)
	cache.Default.PutDuration(key, info, 10*time.Minute)
	return info, nil
}

func parseSubject(m map[string]interface{}) *model.BgmInfo {
	info := &model.BgmInfo{
		ID:       strval(m["id"]),
		URL:      "https://bgm.tv/subject/" + strval(m["id"]),
		Name:     strval(m["name"]),
		NameCn:   strval(m["name_cn"]),
		Eps:      intval(m["eps"]),
		Platform: strval(m["platform"]),
	}
	if dateStr, ok := m["date"].(string); ok && dateStr != "" {
		if t, err := time.ParseInLocation("2006-01-02", dateStr, model.Loc()); err == nil {
			info.Date = model.DateTime(t)
		}
	}
	if images, ok := m["images"].(map[string]interface{}); ok {
		info.Images = model.BgmImages{
			Small:  strval(images["small"]),
			Grid:   strval(images["grid"]),
			Large:  strval(images["large"]),
			Medium: strval(images["medium"]),
			Common: strval(images["common"]),
		}
	}
	if rating, ok := m["rating"].(map[string]interface{}); ok {
		info.Rating.Rank = intval(rating["rank"])
		info.Rating.Score = floatval(rating["score"])
		info.Rating.Total = intval(rating["total"])
	}
	if tags, ok := m["tags"].([]interface{}); ok {
		for _, t := range tags {
			tm, ok := t.(map[string]interface{})
			if !ok {
				continue
			}
			info.Tags = append(info.Tags, model.BgmTag{
				Name:      strval(tm["name"]),
				Count:     intval(tm["count"]),
				TotalCont: intval(tm["totalCont"]),
			})
		}
	}
	return info
}

// GetEpisodes fetches the episode list for a subject.
func GetEpisodes(subjectId string, typ int) []*model.BgmEpisode {
	resp, err := bgmGet(fmt.Sprintf("/v0/episodes?subject_id=%s&type=%d&limit=100&offset=0", subjectId, typ))
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil
	}
	var body struct {
		Data []model.BgmEpisode `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil
	}
	out := make([]*model.BgmEpisode, 0, len(body.Data))
	for i := range body.Data {
		out = append(out, &body.Data[i])
	}
	return out
}

// GetEpisodeTitleMap returns episode -> title maps for an ani.
func GetEpisodeTitleMap(ani *model.Ani) (map[int]string, map[int]string) {
	cn := map[int]string{}
	jp := map[int]string{}
	subjectId := GetSubjectId(ani)
	if subjectId == "" {
		return cn, jp
	}
	key := "BGM_eps:" + subjectId
	var eps []*model.BgmEpisode
	if v, ok := cache.Default.Get(key); ok {
		eps = v.([]*model.BgmEpisode)
	} else {
		eps = GetEpisodes(subjectId, 0)
		if eps == nil {
			eps = GetEpisodes(subjectId, 1)
		}
		cache.Default.PutDuration(key, eps, 5*time.Minute)
	}
	for _, e := range eps {
		i := int(e.Sort)
		if e.Sort == 0 {
			i = int(e.Ep)
		}
		cn[i] = e.NameCn
		jp[i] = e.Name
	}
	return cn, jp
}

// Me returns the authenticated user.
func Me() (*model.BgmMe, error) {
	resp, err := bgmGet("/v0/me")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.New("bgm http " + resp.Status)
	}
	var m map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return nil, err
	}
	me := &model.BgmMe{
		ID:       intval(m["id"]),
		Sign:     strval(m["sign"]),
		URL:      strval(m["url"]),
		Username: strval(m["username"]),
		Nickname: strval(m["nickname"]),
		UserGroup: intval(m["user_group"]),
		Email:     strval(m["email"]),
		TimeOffset: intval(m["time_offset"]),
		ExpiresDays: intval(m["expires_days"]),
	}
	if avatar, ok := m["avatar"].(map[string]interface{}); ok {
		me.Avatar = model.BgmAvatar{
			Large:  strval(avatar["large"]),
			Medium: strval(avatar["medium"]),
			Small:  strval(avatar["small"]),
		}
	}
	return me, nil
}

// GetRate fetches the current collection score for a subject.
func GetRate(subjectId string) (int, error) {
	me, err := Me()
	if err != nil {
		return 0, err
	}
	resp, err := bgmGet(fmt.Sprintf("/v0/users/%s/collections/%s", url.PathEscape(me.Username), subjectId))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, errors.New("bgm http " + resp.Status)
	}
	var m map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return 0, err
	}
	rating := m["rating"]
	return int(floatval(rating)), nil
}

// SetRate updates the collection score.
func SetRate(subjectId string, score float64) error {
	me, err := Me()
	if err != nil {
		return err
	}
	body := map[string]interface{}{"score": score}
	resp, err := bgmPostJSON(fmt.Sprintf("/v0/users/-/collections/%s", subjectId), body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errors.New("bgm http " + resp.Status)
	}
	_ = me
	return nil
}

// ExchangeCode exchanges an OAuth authorization code for tokens.
func ExchangeCode(code string) error {
	cfg := config.Get()
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", cfg.BgmAppID)
	form.Set("client_secret", cfg.BgmAppSecret)
	form.Set("code", code)
	form.Set("redirect_uri", cfg.BgmRedirectUri)
	resp, err := bgmPostForm("https://bgm.tv/oauth/access_token", form)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	access := strval(m["access_token"])
	refresh := strval(m["refresh_token"])
	if access == "" {
		return errors.New("oauth failed: " + string(b))
	}
	cfg.BgmToken = access
	if refresh != "" {
		cfg.BgmRefreshToken = refresh
	}
	return config.Sync()
}

// RefreshToken exchanges the refresh token for a new access token.
func RefreshToken() error {
	cfg := config.Get()
	if cfg.BgmTokenType == "INPUT" {
		return nil
	}
	if cfg.BgmRefreshToken == "" {
		return errors.New("no bgmRefreshToken")
	}
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", cfg.BgmAppID)
	form.Set("client_secret", cfg.BgmAppSecret)
	form.Set("refresh_token", cfg.BgmRefreshToken)
	form.Set("redirect_uri", cfg.BgmRedirectUri)
	resp, err := bgmPostForm("https://bgm.tv/oauth/access_token", form)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	access := strval(m["access_token"])
	refresh := strval(m["refresh_token"])
	if access == "" {
		return errors.New("oauth failed: " + string(b))
	}
	cfg.BgmToken = access
	if refresh != "" {
		cfg.BgmRefreshToken = refresh
	}
	return config.Sync()
}

func strval(v interface{}) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case int:
		return strconv.Itoa(t)
	case json.Number:
		return t.String()
	}
	return fmt.Sprintf("%v", v)
}

func intval(v interface{}) int {
	f := floatval(v)
	return int(f)
}

func floatval(v interface{}) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case float32:
		return float64(t)
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case string:
		f, _ := strconv.ParseFloat(t, 64)
		return f
	case json.Number:
		f, _ := t.Float64()
		return f
	}
	return 0
}