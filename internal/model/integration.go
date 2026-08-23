package model

// Mikan is the search results page model.
type Mikan struct {
	Seasons   []MikanSeason `json:"seasons"`
	Weeks     []MikanWeek   `json:"weeks"`
	TotalItem int           `json:"totalItem"`
}

// MikanSeason describes a selectable season.
type MikanSeason struct {
	Year       int    `json:"year"`
	Season     string `json:"season"`
	SeasonLabel string `json:"seasonLabel"`
	Select     bool   `json:"select"`
}

// MikanWeek groups MikanInfo by week label.
type MikanWeek struct {
	WeekLabel string       `json:"weekLabel"`
	Items     []MikanInfo  `json:"items"`
}

// MikanInfo is one anime entry in Mikan listings.
type MikanInfo struct {
	BgmId   int          `json:"bgmId"`
	Cover   string       `json:"cover"`
	URL     string       `json:"url"`
	Exists  bool         `json:"exists"`
	Score   float64      `json:"score"`
	Title   string       `json:"title"`
	BgmUrl  string       `json:"bgmUrl"`
	Groups  []MikanGroup `json:"groups"`
}

// MikanGroup is one fansub group with its RSS.
type MikanGroup struct {
	SubgroupId string      `json:"subgroupId"`
	Label      string      `json:"label"`
	Rss        string      `json:"rss"`
	BgmUrl     string      `json:"bgmUrl"`
	UpdateDay  string      `json:"updateDay"`
	Items      []MikanItem `json:"items"`
	GroupRegex GroupRegex  `json:"groupRegex"`
}

// MikanItem is a release item on a group page.
type MikanItem struct {
	Title      string   `json:"title"`
	Magnet     string   `json:"magnet"`
	Size       int64    `json:"size"`
	FormatSize string   `json:"formatSize"`
	CreatedAt  DateTime `json:"createdAt"`
	Torrent    string   `json:"torrent"`
}

// BgmInfo is a Bangumi subject.
type BgmInfo struct {
	ID       string        `json:"id"`
	URL      string        `json:"url"`
	Name     string        `json:"name"`
	NameCn   string        `json:"nameCn"`
	Eps      int           `json:"eps"`
	Date     DateTime      `json:"date"`
	Images   BgmImages     `json:"images"`
	Season   int           `json:"season"`
	Platform string        `json:"platform"`
	Tags     []BgmTag      `json:"tags"`
	Infobox  []interface{} `json:"infobox"`
	Rating   BgmRating     `json:"rating"`
}

// BgmImages is the image url set.
type BgmImages struct {
	Small  string `json:"small"`
	Grid   string `json:"grid"`
	Large  string `json:"large"`
	Medium string `json:"medium"`
	Common string `json:"common"`
}

// BgmTag is a subject tag.
type BgmTag struct {
	Name      string `json:"name"`
	Count     int    `json:"count"`
	TotalCont int    `json:"totalCont"`
}

// BgmRating is subject rating info.
type BgmRating struct {
	Rank  int            `json:"rank"`
	Score float64        `json:"score"`
	Total int            `json:"total"`
	Count map[string]int `json:"count"`
}

// BgmEpisodes is the episode list response from bangumi.
type BgmEpisodes struct {
	Data   []BgmEpisode `json:"data"`
	Total  int          `json:"total"`
	Limit  int          `json:"limit"`
	Offset int          `json:"offset"`
}

// BgmEpisode is one episode.
type BgmEpisode struct {
	AirDate        Date    `json:"airdate"`
	Name           string  `json:"name"`
	NameCn         string  `json:"nameCn"`
	Duration       string  `json:"duration"`
	Desc           string  `json:"desc"`
	Ep             float64 `json:"ep"`
	Sort           float64 `json:"sort"`
	ID             int     `json:"id"`
	SubjectId      int     `json:"subjectId"`
	Comment        int     `json:"comment"`
	Type           int     `json:"type"`
	Disc           int     `json:"disc"`
	DurationSeconds int    `json:"durationSeconds"`
}

// BgmMe is the authenticated bangumi user info.
type BgmMe struct {
	Avatar       BgmAvatar `json:"avatar"`
	ID           int       `json:"id"`
	Sign         string    `json:"sign"`
	URL          string    `json:"url"`
	Username     string    `json:"username"`
	Nickname     string    `json:"nickname"`
	UserGroup    int       `json:"userGroup"`
	RegTime      DateTime  `json:"regTime"`
	Email        string    `json:"email"`
	TimeOffset   int       `json:"timeOffset"`
	ExpiresDays  int       `json:"expiresDays"`
}

// BgmAvatar is the user avatar set.
type BgmAvatar struct {
	Large  string `json:"large"`
	Medium string `json:"medium"`
	Small  string `json:"small"`
}

// EmbyViews is a media library view.
type EmbyViews struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// EmbyWebHook is an incoming emby webhook payload (accepts TitleCase alternates).
type EmbyWebHook struct {
	Title        string         `json:"title"`
	Description  string         `json:"description"`
	Date         string         `json:"date"`
	Event        string         `json:"event"`
	Severity     string         `json:"severity"`
	User         *EmbyUser      `json:"user"`
	Server       *EmbyServer    `json:"server"`
	Item         *EmbyItem      `json:"item"`
	PlaybackInfo *EmbyPlayback  `json:"playbackInfo"`
}

// EmbyUser is the webhook user.
type EmbyUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// EmbyServer is the webhook server.
type EmbyServer struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

// EmbyItem is the webhook item.
type EmbyItem struct {
	Path       string `json:"path"`
	SeriesName string `json:"seriesName"`
	FileName   string `json:"fileName"`
}

// EmbyPlayback is the playback info.
type EmbyPlayback struct {
	PlayedToCompletion bool `json:"playedToCompletion"`
}

// PlayItem is a playable file entry for the web player.
type PlayItem struct {
	Title      string           `json:"title"`
	Filename   string           `json:"filename"`
	Name       string           `json:"name"`
	LastModify int64            `json:"lastModify"`
	Episode    float64          `json:"episode"`
	FormatSize string           `json:"formatSize"`
	ExtName    string           `json:"extName"`
	Subtitles  []PlaySubtitle   `json:"subtitles"`
}

// PlaySubtitle is a subtitle track for a play item.
type PlaySubtitle struct {
	HTML    string `json:"html"`
	Name    string `json:"name"`
	URL     string `json:"url"`
	Content string `json:"content"`
	Type    string `json:"type"`
}

// ThemoviedbDTO is the input for getThemoviedbName.
type ThemoviedbDTO struct {
	TmdbId string `json:"tmdbId"`
	Title  string `json:"title"`
	Ova    bool   `json:"ova"`
}

// ThemoviedbVO is the response for getThemoviedbName.
type ThemoviedbVO struct {
	ThemoviedbName string `json:"themoviedbName"`
	Tmdb           *Tmdb  `json:"tmdb"`
}

// TmdbGroup is one TMDB group/season entry.
type TmdbGroup struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	GroupId     string `json:"groupId"`
	Description string `json:"description"`
	// additional passthrough fields for the frontend
	PosterPath string `json:"posterPath"`
}

// IdDTO is a generic id body.
type IdDTO struct {
	ID string `json:"id"`
}

// RssToAniDTO is the input for rssToAni.
type RssToAniDTO struct {
	URL      string `json:"url"`
	Type     string `json:"type"`
	BgmUrl   string `json:"bgmUrl"`
	Subgroup string `json:"subgroup"`
	Enable   bool   `json:"enable"`
}

// ImportAniDataDTO is the input for importAni.
type ImportAniDataDTO struct {
	Filename string `json:"filename"`
	AniList  []*Ani `json:"aniList"`
	Conflict string `json:"conflict"` // REPLACE | SKIP
}

// AniBTQueryDTO is the input for aniBT search.
type AniBTQueryDTO struct {
	Season string `json:"season"`
	BgmUrl string `json:"bgmUrl"`
	Title  string `json:"title"`
}

// AniBT is the ani-bt search result.
type AniBT struct {
	CurrentSeason   string       `json:"currentSeason"`
	RequestedSeason string       `json:"requestedSeason"`
	AvailableSeasons []string    `json:"availableSeasons"`
	ByWeekday       []AniBTByWeekday `json:"byWeekday"`
}

// AniBTByWeekday groups AniBT anime by weekday.
type AniBTByWeekday struct {
	Animes      []AniBTAnime `json:"animes"`
	Weekday     int          `json:"weekday"`
	WeekdayLabel string      `json:"weekdayLabel"`
}

// AniBTAnime is one anime entry.
type AniBTAnime struct {
	AnimeId         string       `json:"animeId"`
	BgmId           StrID        `json:"bgmId"`
	Cover           string       `json:"cover"`
	Rating          float64      `json:"rating"`
	Title           AniBTTitle   `json:"title"`
	Format          string       `json:"format"`
	Exists          bool         `json:"exists"`
	RssReleaseCount int          `json:"rssReleaseCount"`
}

// AniBTTitle is the multi-language title set.
type AniBTTitle struct {
	Chinese           string `json:"chinese"`
	ChineseTraditional string `json:"chineseTraditional"`
	English           string `json:"english"`
	Primary           string `json:"primary"`
	Romaji            string `json:"romaji"`
}

// AniBTGroup is one ani-bt group page.
type AniBTGroup struct {
	BgmId        StrID       `json:"bgmId"`
	GroupId      StrID       `json:"groupId"`
	Slug         string      `json:"slug"`
	Name         string      `json:"name"`
	Status       string      `json:"status"`
	LastUpdatedAt int64      `json:"lastUpdatedAt"`
	Items        []AniBTItem `json:"items"`
	Rss          string      `json:"rss"`
	GroupRegex   GroupRegex  `json:"groupRegex"`
}

// AniBTItem is one release item in an ani-bt group.
type AniBTItem struct {
	EpisodeKey string   `json:"episodeKey"`
	Language   []string `json:"language"`
	Magnet     string   `json:"magnet"`
	PublishedAt int64   `json:"publishedAt"`
	ReleaseId  string   `json:"releaseId"`
	Resolution string   `json:"resolution"`
	Subtitle   string   `json:"subtitle"`
	Title      string   `json:"title"`
	Size       int64    `json:"size"`
	FormatSize string   `json:"formatSize"`
}

// AnimeGarden is the anime-garden week listing.
type AnimeGarden struct {
	WeekLabel string          `json:"weekLabel"`
	Subjects  []AnimeGardenSubject `json:"subjects"`
}

// AnimeGardenSubject is one subject entry.
type AnimeGardenSubject struct {
	ID        StrID     `json:"id"`
	Name      string    `json:"name"`
	Keywords  []string  `json:"keywords"`
	ActivedAt DateTime  `json:"activedAt"`
	IsArchived bool     `json:"isArchived"`
	WeekLabel string    `json:"weekLabel"`
	Exists    bool      `json:"exists"`
	Score     float64   `json:"score"`
	Cover     string    `json:"cover"`
}

// AnimeGardenGroup is one anime-garden group page.
type AnimeGardenGroup struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	LastUpdatedAt DateTime `json:"lastUpdatedAt"`
	Items        []AnimeGardenItem `json:"items"`
	Rss          string   `json:"rss"`
	BgmId        string   `json:"bgmId"`
	GroupRegex   GroupRegex `json:"groupRegex"`
}

// AnimeGardenItem is one release item.
type AnimeGardenItem struct {
	ID         StrID                   `json:"id"`
	Provider   string                  `json:"provider"`
	ProviderId string                  `json:"providerId"`
	Title      string                  `json:"title"`
	Href       string                  `json:"href"`
	Type       string                  `json:"type"`
	Magnet     string                  `json:"magnet"`
	Size       int64                   `json:"size"`
	FormatSize string                  `json:"formatSize"`
	CreatedAt  DateTime                `json:"createdAt"`
	FetchedAt  DateTime                `json:"fetchedAt"`
	SubjectId  StrID                   `json:"subjectId"`
	Publisher  *AnimeGardenPublisher   `json:"publisher"`
	Fansub     *AnimeGardenFansub      `json:"fansub"`
}

// AnimeGardenPublisher is the publisher info.
type AnimeGardenPublisher struct {
	ID     StrID  `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

// AnimeGardenFansub is the fansub info.
type AnimeGardenFansub struct {
	ID     StrID  `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}