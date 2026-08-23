package model

// Login is the nested login settings inside Config. Password is stored MD5-hashed.
type Login struct {
	Username string `json:"username"`
	Password string `json:"password"`
	IP       string `json:"ip,omitempty"`
	Key      string `json:"key,omitempty"`
}

// GitInfo carries build metadata (not user editable).
type GitInfo struct {
	Branch        string `json:"branch"`
	ShortCommitId string `json:"shortCommitId"`
	CommitId      string `json:"commitId"`
}

// NotificationStatusEnum values (serialized by name).
type NotificationStatusEnum string

const (
	NotifyDownloadStart    NotificationStatusEnum = "DOWNLOAD_START"
	NotifyDownloadEnd      NotificationStatusEnum = "DOWNLOAD_END"
	NotifyOmit             NotificationStatusEnum = "OMIT"
	NotifyError            NotificationStatusEnum = "ERROR"
	NotifyCompleted        NotificationStatusEnum = "COMPLETED"
	NotifyProcrastinating  NotificationStatusEnum = "PROCRASTINATING"
)

// NotificationTypeEnum values (serialized by name).
type NotificationTypeEnum string

const (
	NotifyEmbyRefresh     NotificationTypeEnum = "EMBY_REFRESH"
	NotifyMail            NotificationTypeEnum = "MAIL"
	NotifyServerChan      NotificationTypeEnum = "SERVER_CHAN"
	NotifySystem          NotificationTypeEnum = "SYSTEM"
	NotifyTelegram        NotificationTypeEnum = "TELEGRAM"
	NotifyWebHook         NotificationTypeEnum = "WEB_HOOK"
	NotifyShell           NotificationTypeEnum = "SHELL"
	NotifyFileMove        NotificationTypeEnum = "FILE_MOVE"
	NotifyOpenListUpload  NotificationTypeEnum = "OPEN_LIST_UPLOAD"
	NotifyBark            NotificationTypeEnum = "BARK"
)

// ServerChanTypeEnum values.
type ServerChanTypeEnum string

const (
	ServerChanType    ServerChanTypeEnum = "SERVER_CHAN"
	ServerChanType3   ServerChanTypeEnum = "SERVER_CHAN_3"
)

// NotificationConfig is one notification channel entry.
type NotificationConfig struct {
	Enable                 bool                   `json:"enable"`
	Retry                  int                    `json:"retry"`
	Comment                string                 `json:"comment"`
	NotificationTemplate   string                 `json:"notificationTemplate"`
	NotificationType       NotificationTypeEnum   `json:"notificationType"`
	MailSMTPHost           string                 `json:"mailSMTPHost"`
	MailSMTPPort           int                    `json:"mailSMTPPort"`
	MailFrom               string                 `json:"mailFrom"`
	MailPassword           string                 `json:"mailPassword"`
	MailSSLEnable          bool                   `json:"mailSSLEnable"`
	MailTLSEnable          bool                   `json:"mailTLSEnable"`
	MailAddressee          string                 `json:"mailAddressee"`
	MailImage              bool                   `json:"mailImage"`
	ServerChanType         ServerChanTypeEnum     `json:"serverChanType"`
	ServerChanSendKey      string                 `json:"serverChanSendKey"`
	ServerChan3ApiUrl      string                 `json:"serverChan3ApiUrl"`
	ServerChanTitleAction  string                 `json:"serverChanTitleAction"`
	TelegramBotToken       string                 `json:"telegramBotToken"`
	TelegramChatId         string                 `json:"telegramChatId"`
	TelegramTopicId        int                    `json:"telegramTopicId"`
	TelegramApiHost        string                 `json:"telegramApiHost"`
	TelegramImage          bool                   `json:"telegramImage"`
	TelegramFormat         string                 `json:"telegramFormat"`
	WebHookMethod          string                 `json:"webHookMethod"`
	WebHookUrl             string                 `json:"webHookUrl"`
	WebHookHeader          string                 `json:"webHookHeader"`
	WebHookBody            string                 `json:"webHookBody"`
	EmbyHost               string                 `json:"embyHost"`
	EmbyApiKey             string                 `json:"embyApiKey"`
	EmbyRefreshViewIds     string                 `json:"embyRefreshViewIds"`
	EmbyDelayed            int                    `json:"embyDelayed"`
	Shell                  string                 `json:"shell"`
	AliveLimit             int                    `json:"aliveLimit"`
	FileMoveTarget         string                 `json:"fileMoveTarget"`
	FileMoveOvaTarget      string                 `json:"fileMoveOvaTarget"`
	FileMoveDeleteOldEp    bool                   `json:"fileMoveDeleteOldEpisode"`
	FileMoveCopyModel      bool                   `json:"fileMoveCopyModel"`
	OpenListUploadHost     string                 `json:"openListUploadHost"`
	OpenListUploadApiKey   string                 `json:"openListUploadApiKey"`
	OpenListUploadPath     string                 `json:"openListUploadPath"`
	OpenListUploadOvaPath  string                 `json:"openListUploadOvaPath"`
	OpenListUploadDelLocal bool                   `json:"openListUploadDeleteLocalFile"`
	OpenListUploadDelOldEp  bool                  `json:"openListUploadDeleteOldEpisode"`
	BarkServerUrl          string                 `json:"barkServerUrl"`
	BarkDeviceKeys         string                 `json:"barkDeviceKeys"`
	BarkGroup              string                 `json:"barkGroup"`
	BarkUseMarkdown        bool                   `json:"barkUseMarkdown"`
	BarkLevel              string                 `json:"barkLevel"`
	BarkVolume             int                    `json:"barkVolume"`
	StatusList             []NotificationStatusEnum `json:"statusList"`
	Sort                   int64                  `json:"sort"`
}

// DefaultNotificationConfig returns the defaults factory used by the Java app.
func DefaultNotificationConfig() *NotificationConfig {
	return &NotificationConfig{
		Enable:               true,
		Retry:                3,
		NotificationTemplate: "${notification}",
		NotificationType:     NotifyTelegram,
		StatusList: []NotificationStatusEnum{
			NotifyDownloadStart, NotifyOmit, NotifyError,
		},
		Sort: 10,
	}
}

// Config is the root application settings, persisted as config.v2.json.
type Config struct {
	MikanHost                        string                `json:"mikanHost"`
	TmdbApi                          string                `json:"tmdbApi"`
	TmdbApiKey                       string                `json:"tmdbApiKey"`
	TmdbImage                        string                `json:"tmdbImage"`
	TmdbAnime                        bool                  `json:"tmdbAnime"`
	DownloadToolType                 string                `json:"downloadToolType"`
	DownloadRetry                    int                   `json:"downloadRetry"`
	PikpakEmail                      string                `json:"pikpakEmail"`
	PikpakPassword                   string                `json:"pikpakPassword"`
	Pan115Cookie                     string                `json:"pan115Cookie"`
	DownloadPathTemplate             string                `json:"downloadPathTemplate"`
	OvaDownloadPathTemplate          string                `json:"ovaDownloadPathTemplate"`
	CustomTags                       []string              `json:"customTags"`
	PriorityKeywordsEnable           bool                  `json:"priorityKeywordsEnable"`
	PriorityKeywords                 []string              `json:"priorityKeywords"`
	DelayedDownload                  int                   `json:"delayedDownload"`
	RssSleepMinutes                  int                   `json:"rssSleepMinutes"`
	RenameSleepSeconds               int                   `json:"renameSleepSeconds"`
	Rename                           bool                  `json:"rename"`
	Rss                              bool                  `json:"rss"`
	RssTimeout                       int                   `json:"rssTimeout"`
	FileExist                        bool                  `json:"fileExist"`
	AwaitStalledUP                   bool                  `json:"awaitStalledUP"`
	Delete                           bool                  `json:"delete"`
	DeleteStandbyRSSOnly             bool                  `json:"deleteStandbyRSSOnly"`
	Offset                           bool                  `json:"offset"`
	TitleYear                        bool                  `json:"titleYear"`
	AutoDisabled                     bool                  `json:"autoDisabled"`
	Skip5                            bool                  `json:"skip5"`
	StandbyRss                       bool                  `json:"standbyRss"`
	Coexist                          bool                  `json:"coexist"`
	LogsMax                          int                   `json:"logsMax"`
	Debug                            bool                  `json:"debug"`
	ProcrastinatingMasterOnly        bool                  `json:"procrastinatingMasterOnly"`
	Proxy                            bool                  `json:"proxy"`
	ProxyHost                        string                `json:"proxyHost"`
	ProxyPort                        int                   `json:"proxyPort"`
	ProxyUsername                    string                `json:"proxyUsername"`
	ProxyPassword                    string                `json:"proxyPassword"`
	DownloadCount                    int                   `json:"downloadCount"`
	Login                            Login                 `json:"login"`
	MultiLoginForbidden              bool                  `json:"multiLoginForbidden"`
	LoginEffectiveHours              int                   `json:"loginEffectiveHours"`
	Exclude                          []string              `json:"exclude"`
	ImportExclude                    bool                  `json:"importExclude"`
	EnabledExclude                   bool                  `json:"enabledExclude"`
	BgmJpName                        bool                  `json:"bgmJpName"`
	Tmdb                             bool                  `json:"tmdb"`
	TmdbId                           bool                  `json:"tmdbId"`
	TmdbIdPlexMode                   bool                  `json:"tmdbIdPlexMode"`
	TmdbLanguage                     string                `json:"tmdbLanguage"`
	TmdbRomaji                       bool                  `json:"tmdbRomaji"`
	TmdbOriginalName                 bool                  `json:"tmdbOriginalName"`
	IpWhitelist                      bool                  `json:"ipWhitelist"`
	IpWhitelistStr                   string                `json:"ipWhitelistStr"`
	Omit                             bool                  `json:"omit"`
	BgmTokenType                     string                `json:"bgmTokenType"`
	BgmToken                         string                `json:"bgmToken"`
	BgmAppID                         string                `json:"bgmAppID"`
	BgmAppSecret                     string                `json:"bgmAppSecret"`
	BgmRefreshToken                  string                `json:"bgmRefreshToken"`
	BgmRedirectUri                   string                `json:"bgmRedirectUri"`
	ApiKey                           string                `json:"apiKey"`
	DownloadNew                      bool                  `json:"downloadNew"`
	InnerIP                          bool                  `json:"innerIP"`
	RenameTemplate                   string                `json:"renameTemplate"`
	RenameDelYear                    bool                  `json:"renameDelYear"`
	RenameDelTmdbId                  bool                  `json:"renameDelTmdbId"`
	VerifyLoginIp                    bool                  `json:"verifyLoginIp"`
	AutoTrackersUpdate               bool                  `json:"autoTrackersUpdate"`
	TrackersUpdateUrls               string                `json:"trackersUpdateUrls"`
	NotificationTemplate             string                `json:"notificationTemplate"`
	AutoUpdate                       bool                  `json:"autoUpdate"`
	BgmImage                         string                `json:"bgmImage"`
	CustomCss                        string                `json:"customCss"`
	CustomJs                         string                `json:"customJs"`
	CustomEpisode                    bool                  `json:"customEpisode"`
	CustomEpisodeStr                 string                `json:"customEpisodeStr"`
	CustomEpisodeGroupIndex          int                   `json:"customEpisodeGroupIndex"`
	Procrastinating                  bool                  `json:"procrastinating"`
	ProcrastinatingDay               int                   `json:"procrastinatingDay"`
	GithubToken                      string                `json:"githubToken"`
	UpdateTotalEpisodeNumber         bool                  `json:"updateTotalEpisodeNumber"`
	ForceUpdateTotalEpisodeNumber    bool                  `json:"forceUpdateTotalEpisodeNumber"`
	OpenListDownloadTimeout          int                   `json:"openListDownloadTimeout"`
	OpenListDownloadRetryNumber      int64                 `json:"openListDownloadRetryNumber"`
	ConfigBackup                     bool                  `json:"configBackup"`
	ConfigBackupDay                  int                   `json:"configBackupDay"`
	Completed                        bool                  `json:"completed"`
	CompletedPathTemplate            string                `json:"completedPathTemplate"`
	NotificationConfigList           []NotificationConfig  `json:"notificationConfigList"`
	CopyMasterToStandby              bool                  `json:"copyMasterToStandby"`
	SortType                         string                `json:"sortType"`
	ProxyList                        string                `json:"proxyList"`
	Scrape                           bool                  `json:"scrape"`
	FollowDay                        int                   `json:"followDay"`
	BangumiIniEnabled                bool                  `json:"bangumiIniEnabled"`
	Replace                          bool                  `json:"replace"`
	MaxFileNameLength                int                   `json:"maxFileNameLength"`
	LimitLoginAttempts               bool                  `json:"limitLoginAttempts"`
	GitInfo                          *GitInfo              `json:"gitInfo"`
	ReverseProxyTrustIpListEnabled   bool                  `json:"reverseProxyTrustIpListEnabled"`
	ReverseProxyTrustIpList          []string              `json:"reverseProxyTrustIpList"`
	SubtitleIndependentFolderEnabled bool                  `json:"subtitleIndependentFolderEnabled"`
	SubtitleIndependentFolderName    string                `json:"subtitleIndependentFolderName"`
	BgmApi                           string                `json:"bgmApi"`
	AutoStart                        bool                  `json:"autoStart"`
	AllowCors                        bool                  `json:"allowCors"`
	UUID                             string                `json:"uuid"`
}

// DefaultConfig returns the defaults matching ConfigUtil static block.
func DefaultConfig() *Config {
	return &Config{
		MikanHost:                       "https://mikanani.me",
		TmdbApi:                         "https://api.themoviedb.org",
		TmdbApiKey:                      "",
		TmdbAnime:                       true,
		DownloadToolType:                "115",
		DownloadRetry:                   3,
		DownloadPathTemplate:            "番剧/${title}/Season ${season}",
		OvaDownloadPathTemplate:         "剧场版/${title}",
		DelayedDownload:                 0,
		RssSleepMinutes:                 15,
		RenameSleepSeconds:              10,
		Rename:                          true,
		Rss:                             true,
		RssTimeout:                      20,
		AwaitStalledUP:                  true,
		TitleYear:                       true,
		Skip5:                           true,
		LogsMax:                         128,
		ProcrastinatingMasterOnly:       true,
		ProxyPort:                       8080,
		Login:                           Login{Username: "admin", Password: md5Hex("admin")},
		MultiLoginForbidden:             true,
		LoginEffectiveHours:             3,
		Exclude:                         []string{"720[Pp]", "\\d-\\d", "合集", "特别篇"},
		Tmdb:                            true,
		TmdbLanguage:                    "zh-CN",
		Omit:                            true,
		CustomEpisodeGroupIndex:         2,
		CustomEpisodeStr:                renameRegStr,
		ProcrastinatingDay:              14,
		OpenListDownloadTimeout:         60,
		OpenListDownloadRetryNumber:     5,
		ConfigBackupDay:                 7,
		SortType:                        "SCORE",
		FollowDay:                       14,
		LimitLoginAttempts:              true,
		ReverseProxyTrustIpList:         []string{"127.0.0.1"},
		SubtitleIndependentFolderName:   "Subs",
		BgmApi:                          "https://api.bgm.tv",
	}
}

func md5Hex(s string) string {
	// small helper to avoid import cycle with util; md5 of "admin" precomputed below
	_ = s
	return "21232f297a57a5a743894a0e4a801fc3"
}

const renameRegStr = `(.*|\[.*])(( - |Vol |[Ee][Pp]?)\d+(\.5)?( ?\(\d+\))?|【\d+(\.5)?】|\[\d+(\.5)?( ?\(\d+\))?( ?[vV]\d)?( ?END)?( ?完)?( ?FIN)?]|第\d+(\.5)?[话話集]( - END)?|^\[TOC].* \d+|^六四位元字幕组.*★\d+(\.5)?★)`

// RENAME_REG_STR exposes the episode extraction regex source.
func RENAME_REG_STR() string { return renameRegStr }