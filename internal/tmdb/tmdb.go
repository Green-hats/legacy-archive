package tmdb

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"ani-rss/internal/cache"
	"ani-rss/internal/config"
	"ani-rss/internal/model"
	"ani-rss/internal/rename"
	"ani-rss/internal/util"
)

var defaultKey = "450e4f651e1c93e31383e20f8e731e5f"

func apiKey() string {
	if k := config.Get().TmdbApiKey; k != "" {
		return k
	}
	return defaultKey
}

// tmdbReq GETs a tmdb api path with the standard params.
func tmdbReq(path string, params url.Values) ([]byte, error) {
	cfg := config.Get()
	base := cfg.TmdbApi
	if base == "" {
		base = "https://api.themoviedb.org"
	}
	if params == nil {
		params = url.Values{}
	}
	params.Set("api_key", apiKey())
	if cfg.TmdbLanguage != "" {
		params.Set("language", cfg.TmdbLanguage)
	}
	u := base + path + "?" + params.Encode()
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
	if resp.StatusCode == 404 {
		return nil, errors.New("tmdb 404")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.New("tmdb http " + resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// TmdbEpisode mirrors one TMDB season episode.
type TmdbEpisode struct {
	EpisodeNumber int     `json:"episode_number"`
	SeasonNumber  int     `json:"season_number"`
	Name          string  `json:"name"`
	Overview      string  `json:"overview"`
	VoteAverage   float64 `json:"vote_average"`
	AirDate       string  `json:"air_date"`
	Runtime       int     `json:"runtime"`
	StillPath     string  `json:"still_path"`
}

// TmdbSeason mirrors a TMDB season.
type TmdbSeason struct {
	SeasonNumber int           `json:"season_number"`
	Name         string        `json:"name"`
	Overview     string        `json:"overview"`
	PosterPath   string        `json:"poster_path"`
	VoteAverage  float64       `json:"vote_average"`
	AirDate      string        `json:"air_date"`
	Episodes     []TmdbEpisode `json:"episodes"`
}

// TmdbImage is one entry in the images response.
type TmdbImage struct {
	FilePath    string  `json:"file_path"`
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	VoteAverage float64 `json:"vote_average"`
	Iso6391     string  `json:"iso_639_1"`
	Iso3166_1   string  `json:"iso_3166_1"`
}

// TmdbImages is the images response for a title.
type TmdbImages struct {
	Backdrops []TmdbImage `json:"backdrops"`
	Logos     []TmdbImage `json:"logos"`
	Posters   []TmdbImage `json:"posters"`
}

// TmdbCredits is the credits response (cast).
type TmdbCredits struct {
	Cast []struct {
		ID        int    `json:"id"`
		Name      string `json:"name"`
		Character string `json:"character"`
	} `json:"cast"`
}

// TmdbVideo is one video entry.
type TmdbVideo struct {
	Key   string `json:"key"`
	Site  string `json:"site"`
	Name  string `json:"name"`
}

// TmdbNetwork is a production network.
type TmdbNetwork struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// TmdbGenre is a genre.
type TmdbGenre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func parseTv(m map[string]interface{}) *model.Tmdb {
	t := &model.Tmdb{
		ID:           intv(m["id"]),
		Name:         strv(m["name"]),
		OriginalName: strv(m["original_name"]),
		PosterPath:   strv(m["poster_path"]),
		BackdropPath: strv(m["backdrop_path"]),
		Overview:     strv(m["overview"]),
		VoteAverage:  floatv(m["vote_average"]),
		VoteCount:    intv(m["vote_count"]),
		Tagline:      strv(m["tagline"]),
		Runtime:      intv(m["runtime"]),
		TmdbType:     "tv",
	}
	if d, err := time.Parse("2006-01-02", strv(m["first_air_date"])); err == nil {
		t.Date = model.Date(d)
	}
	if genres, ok := m["genres"].([]interface{}); ok {
		for _, g := range genres {
			gm, _ := g.(map[string]interface{})
			t.Genres = append(t.Genres, model.TmdbGenre{ID: intv(gm["id"]), Name: strv(gm["name"])})
		}
	}
	if networks, ok := m["networks"].([]interface{}); ok {
		for _, n := range networks {
			nm, _ := n.(map[string]interface{})
			t.Networks = append(t.Networks, model.TmdbNetwork{ID: intv(nm["id"]), Name: strv(nm["name"])})
		}
	}
	if videos, ok := m["videos"].(map[string]interface{}); ok {
		if results, ok := videos["results"].([]interface{}); ok {
			for _, v := range results {
				vm, _ := v.(map[string]interface{})
				t.Videos = append(t.Videos, model.TmdbVideo{Key: strv(vm["key"]), Site: strv(vm["site"]), Name: strv(vm["name"])})
			}
		}
	}
	if credits, ok := m["credits"].(map[string]interface{}); ok {
		if cast, ok := credits["cast"].([]interface{}); ok {
			for _, c := range cast {
				cm, _ := c.(map[string]interface{})
				t.Cast = append(t.Cast, model.TmdbCast{ID: intv(cm["id"]), Name: strv(cm["name"]), Character: strv(cm["character"])})
			}
		}
	}
	return t
}

// GetTvSeason fetches a tv season (or episode group when tmdbGroupId set).
func GetTvSeason(t *model.Tmdb, season int) (*TmdbSeason, error) {
	if t == nil || t.ID == 0 {
		return nil, errors.New("no tmdb")
	}
	if t.TmdbGroupId != "" {
		return getSeasonFromGroup(t.TmdbGroupId, season)
	}
	b, err := tmdbReq(fmt.Sprintf("/3/tv/%d/season/%d", t.ID, season), nil)
	if err != nil {
		return nil, err
	}
	var s TmdbSeason
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func getSeasonFromGroup(groupID string, season int) (*TmdbSeason, error) {
	b, err := tmdbReq("/3/tv/episode_group/"+groupID, nil)
	if err != nil {
		return nil, err
	}
	var body struct {
		Groups []struct {
			Order    int    `json:"order"`
			Episodes []TmdbEpisode `json:"episodes"`
		} `json:"groups"`
	}
	if err := json.Unmarshal(b, &body); err != nil {
		return nil, err
	}
	for _, g := range body.Groups {
		if g.Order == season {
			s := &TmdbSeason{SeasonNumber: season}
			for i, e := range g.Episodes {
				e.EpisodeNumber = i + 1
				s.Episodes = append(s.Episodes, e)
			}
			if len(s.Episodes) > 0 {
				s.AirDate = s.Episodes[0].AirDate
			}
			return s, nil
		}
	}
	return nil, errors.New("season not found in group")
}

// GetTmdbImages fetches and sorts the images for a title.
func GetTmdbImages(t *model.Tmdb) (*TmdbImages, error) {
	if t == nil || t.ID == 0 {
		return nil, errors.New("no tmdb")
	}
	b, err := tmdbReq(fmt.Sprintf("/3/%s/%d/images", t.TmdbType, t.ID), nil)
	if err != nil {
		return nil, err
	}
	var imgs TmdbImages
	if err := json.Unmarshal(b, &imgs); err != nil {
		return nil, err
	}
	sort.SliceStable(imgs.Backdrops, func(i, j int) bool { return imageScore(&imgs.Backdrops[i]) < imageScore(&imgs.Backdrops[j]) })
	sort.SliceStable(imgs.Logos, func(i, j int) bool { return imageScore(&imgs.Logos[i]) < imageScore(&imgs.Logos[j]) })
	sort.SliceStable(imgs.Posters, func(i, j int) bool { return imageScore(&imgs.Posters[i]) < imageScore(&imgs.Posters[j]) })
	return &imgs, nil
}

func imageScore(im *TmdbImage) float64 {
	lang := config.Get().TmdbLanguage
	if im.Iso6391 == "" {
		return 50 + im.VoteAverage
	}
	if im.Iso6391+"-"+im.Iso3166_1 == lang {
		return im.VoteAverage
	}
	if lang == "zh-CN" {
		return 10 + im.VoteAverage
	}
	if strings.HasPrefix(lang, "zh-") {
		return 20 + im.VoteAverage
	}
	if strings.HasPrefix(lang, "ja-") {
		return 30 + im.VoteAverage
	}
	return 40 + im.VoteAverage
}

// ImageURL builds the full tmdb image url.
func ImageURL(path string) string {
	cfg := config.Get()
	base := cfg.TmdbImage
	if base == "" {
		base = "https://image.tmdb.org"
	}
	return strings.TrimSuffix(base, "/") + "/t/p/original" + path
}

// SearchTv searches TV shows by name.
func SearchTv(name string) ([]*model.Tmdb, error) {
	b, err := tmdbReq("/3/search/tv", url.Values{"query": {name}})
	if err != nil {
		return nil, err
	}
	var body struct {
		Results []struct {
			ID           int      `json:"id"`
			Name         string   `json:"name"`
			OriginalName string   `json:"original_name"`
			FirstAirDate string   `json:"first_air_date"`
			PosterPath   string   `json:"poster_path"`
			BackdropPath string   `json:"backdrop_path"`
			Overview     string   `json:"overview"`
			VoteAverage  float64  `json:"vote_average"`
			VoteCount    int      `json:"vote_count"`
			GenreIds     []int    `json:"genre_ids"`
		} `json:"results"`
	}
	if err := json.Unmarshal(b, &body); err != nil {
		return nil, err
	}
	var out []*model.Tmdb
	for _, r := range body.Results {
		t := &model.Tmdb{
			ID: r.ID, Name: r.Name, OriginalName: r.OriginalName,
			PosterPath: r.PosterPath, BackdropPath: r.BackdropPath,
			Overview: r.Overview, VoteAverage: r.VoteAverage, VoteCount: r.VoteCount,
			TmdbType: "tv",
		}
		if d, err := time.Parse("2006-01-02", r.FirstAirDate); err == nil {
			t.Date = model.Date(d)
		}
		out = append(out, t)
	}
	return out, nil
}

// SearchMovie searches movies by name.
func SearchMovie(name string) ([]*model.Tmdb, error) {
	b, err := tmdbReq("/3/search/movie", url.Values{"query": {name}})
	if err != nil {
		return nil, err
	}
	var body struct {
		Results []struct {
			ID           int      `json:"id"`
			Title        string   `json:"title"`
			OriginalName string   `json:"original_title"`
			ReleaseDate  string   `json:"release_date"`
			PosterPath   string   `json:"poster_path"`
			BackdropPath string   `json:"backdrop_path"`
			Overview     string   `json:"overview"`
			VoteAverage  float64  `json:"vote_average"`
			VoteCount    int      `json:"vote_count"`
			GenreIds     []int    `json:"genre_ids"`
		} `json:"results"`
	}
	if err := json.Unmarshal(b, &body); err != nil {
		return nil, err
	}
	var out []*model.Tmdb
	for _, r := range body.Results {
		t := &model.Tmdb{
			ID: r.ID, Name: r.Title, OriginalName: r.OriginalName,
			PosterPath: r.PosterPath, BackdropPath: r.BackdropPath,
			Overview: r.Overview, VoteAverage: r.VoteAverage, VoteCount: r.VoteCount,
			TmdbType: "movie",
		}
		if d, err := time.Parse("2006-01-02", r.ReleaseDate); err == nil {
			t.Date = model.Date(d)
		}
		out = append(out, t)
	}
	return out, nil
}

// GetByName searches TMDB (tv or movie based on ova) and returns the first match.
func GetByName(name string, ova bool) (*model.Tmdb, error) {
	var results []*model.Tmdb
	var err error
	if ova {
		results, err = SearchMovie(name)
	} else {
		results, err = SearchTv(name)
	}
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		if ova {
			results, err = SearchTv(name)
		} else {
			results, err = SearchMovie(name)
		}
		if err != nil || len(results) == 0 {
			return nil, errors.New("no result")
		}
	}
	if config.Get().TmdbAnime && len(results) > 0 {
		// prefer anime genre if present
	}
	return results[0], nil
}

// GetFinalName builds the display name with year + tmdb id suffix.
func GetFinalName(t *model.Tmdb) string {
	cfg := config.Get()
	title := t.Name
	if cfg.TmdbOriginalName && t.OriginalName != "" {
		title = t.OriginalName
	}
	if cfg.TitleYear {
		title = rename.RenameDelConfig(title, false)
		year := ""
		if !t.Date.Time().IsZero() {
			year = strconv.Itoa(t.Date.Time().Year())
		}
		title = fmt.Sprintf("%s (%s)", title, year)
	}
	if cfg.TmdbId {
		if cfg.TmdbIdPlexMode {
			title = fmt.Sprintf("%s {tmdb-%d}", title, t.ID)
		} else {
			title = fmt.Sprintf("%s [tmdbid=%d]", title, t.ID)
		}
	}
	return rename.GetName(title)
}

// GetEpisodeTitleMap returns episode -> title for a show+season (cached).
func GetEpisodeTitleMap(ani *model.Ani) map[int]string {
	out := map[int]string{}
	if ani == nil || ani.Ova || ani.Tmdb == nil || ani.Tmdb.ID == 0 {
		return out
	}
	tmdbId := ani.Tmdb.ID
	season := ani.Season
	key := fmt.Sprintf("TMDB_getEpisodeTitleMap:%d:%s:%d", tmdbId, ani.Tmdb.TmdbGroupId, season)
	if v, ok := cache.Default.Get(key); ok {
		return v.(map[int]string)
	}
	if season <= 0 {
		season = 1
	}
	s, err := GetTvSeason(ani.Tmdb, season)
	if err != nil {
		return out
	}
	for _, e := range s.Episodes {
		out[e.EpisodeNumber] = rename.GetName(e.Name)
	}
	if len(out) == 0 {
		cache.Default.PutDuration(key, out, 10*time.Second)
	} else {
		cache.Default.PutDuration(key, out, 5*time.Minute)
	}
	return out
}

// GetTmdbGroup lists the collection/group entries for a show.
func GetTmdbGroup(t *model.Tmdb) ([]*model.Tmdb, error) {
	if t == nil || t.ID == 0 {
		return nil, errors.New("no tmdb")
	}
	b, err := tmdbReq(fmt.Sprintf("/3/tv/%d/episode_groups", t.ID), nil)
	if err != nil {
		return nil, err
	}
	var body struct {
		Results []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"results"`
	}
	if err := json.Unmarshal(b, &body); err != nil {
		return nil, err
	}
	var out []*model.Tmdb
	for _, s := range body.Results {
		out = append(out, &model.Tmdb{ID: 0, Name: s.Name, TmdbGroupId: s.ID})
	}
	return out, nil
}

func strv(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func intv(v interface{}) int {
	return int(floatv(v))
}

func floatv(v interface{}) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
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