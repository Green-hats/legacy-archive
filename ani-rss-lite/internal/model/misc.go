package model

// Item is a single RSS feed item.
type Item struct {
	Title       string   `json:"title"`
	ReName      string   `json:"reName"`
	Torrent     string   `json:"torrent"`
	InfoHash    string   `json:"infoHash"`
	Episode     float64  `json:"episode"`
	FormatSize  string   `json:"formatSize"`
	Length      int64    `json:"length"`
	HasDownloaded bool   `json:"hasDownloaded"`
	Master      bool     `json:"master"`
	Subgroup    string   `json:"subgroup"`
	PubDate     DateTime `json:"pubDate"`
}

// Clone returns a shallow copy of the Item.
func (i *Item) Clone() *Item {
	if i == nil {
		return nil
	}
	c := *i
	return &c
}

// ListAni is the grouped subscription list response.
type ListAni struct {
	ReleaseDateList []string   `json:"releaseDateList"`
	WeekList        []WeekAni  `json:"weekList"`
	Total           int        `json:"total"`
}

// WeekAni groups Ani entries by week label.
type WeekAni struct {
	WeekLabel string `json:"weekLabel"`
	Items     []*Ani `json:"items"`
}

// Log is one in-memory log entry.
type Log struct {
	Message    string `json:"message"`
	Level      string `json:"level"`
	LoggerName string `json:"loggerName"`
	ThreadName string `json:"threadName"`
}

// About carries version/update info for /api/about.
type About struct {
	Version     string   `json:"version"`
	Latest      string   `json:"latest"`
	Update      bool     `json:"update"`
	DownloadURL string   `json:"downloadUrl"`
	SHA256      string   `json:"sha256"`
	Size        int64    `json:"size"`
	FormatSize  string   `json:"formatSize"`
	MarkdownBody string  `json:"markdownBody"`
	Date        DateTime `json:"date"`
}

// GroupRegex holds fansub filter rules for the subgroup picker.
type GroupRegex struct {
	RegexList [][]GroupRegexItem `json:"regexList"`
	Tags      []string           `json:"tags"`
}

// GroupRegexItem is one label+regex pair.
type GroupRegexItem struct {
	Label string `json:"label"`
	Regex string `json:"regex"`
}

// ProxyTest is the result of a proxy connectivity test.
type ProxyTest struct {
	Status int   `json:"status"`
	Time   int64 `json:"time"`
}

// Result is the global JSON response wrapper.
type Result struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	T       int64       `json:"t"`
}

// NewResult builds a success result.
func NewResult(data interface{}) *Result {
	return &Result{Code: 200, Message: "success", Data: data, T: Now().UnixMilli()}
}

// NewError builds an error result (code 500).
func NewError(msg string) *Result {
	return &Result{Code: 500, Message: msg, T: Now().UnixMilli()}
}

// NewResultCode builds a result with custom code+message.
func NewResultCode(code int, msg string) *Result {
	return &Result{Code: code, Message: msg, T: Now().UnixMilli()}
}

// NewMessage builds a success result with a custom message.
func NewMessage(msg string) *Result {
	return &Result{Code: 200, Message: msg, T: Now().UnixMilli()}
}

// TorrentsStateEnum values.
type TorrentsStateEnum string

const (
	StateUnknown          TorrentsStateEnum = "unknown"
	StateForcedDL         TorrentsStateEnum = "forcedDL"
	StateDownloading      TorrentsStateEnum = "downloading"
	StateForcedMetaDL     TorrentsStateEnum = "forcedMetaDL"
	StateMetaDL           TorrentsStateEnum = "metaDL"
	StateStalledDL        TorrentsStateEnum = "stalledDL"
	StateForcedUP         TorrentsStateEnum = "forcedUP"
	StateUploading        TorrentsStateEnum = "uploading"
	StateStalledUP        TorrentsStateEnum = "stalledUP"
	StateCheckingResume   TorrentsStateEnum = "checkingResumeData"
	StateQueuedDL         TorrentsStateEnum = "queuedDL"
	StateQueuedUP         TorrentsStateEnum = "queuedUP"
	StateCheckingUP       TorrentsStateEnum = "checkingUP"
	StateCheckingDL       TorrentsStateEnum = "checkingDL"
	StateStoppedDL        TorrentsStateEnum = "stoppedDL"
	StatePausedDL         TorrentsStateEnum = "pausedDL"
	StateStoppedUP        TorrentsStateEnum = "stoppedUP"
	StatePausedUP         TorrentsStateEnum = "pausedUP"
	StateMoving           TorrentsStateEnum = "moving"
	StateMissingFiles     TorrentsStateEnum = "missingFiles"
	StateError            TorrentsStateEnum = "error"
	StateAllocating       TorrentsStateEnum = "allocating"
)

// TorrentsTagEnum values.
const (
	TagAniRss        = "ani-rss"
	TagRename        = "RENAME"
	TagStandbyRss    = "备用RSS"
	TagDownloadDone  = "下载完成"
)

// TorrentsInfo is a normalized torrent entry from any download client.
type TorrentsInfo struct {
	ID         int64             `json:"id"`
	Hash       string            `json:"hash"`
	Name       string            `json:"name"`
	State      TorrentsStateEnum `json:"state"`
	Category   string            `json:"category"`
	TagList    []string          `json:"tagList"`
	Completed  int64             `json:"completed"`
	Size       int64             `json:"size"`
	Progress   float64           `json:"progress"`
	FormatSize string            `json:"formatSize"`
	SavePath   string            `json:"savePath"`
}

// Finished reports whether the torrent is in a finished state.
func (t *TorrentsInfo) Finished() bool {
	if t == nil {
		return false
	}
	if t.Progress >= 100 {
		return true
	}
	switch t.State {
	case StateQueuedUP, StateUploading, StateStalledUP, StateStoppedUP, StateForcedUP:
		return true
	}
	return false
}

// HasTag reports whether the torrent carries the given tag.
func (t *TorrentsInfo) HasTag(tag string) bool {
	for _, tg := range t.TagList {
		if tg == tag {
			return true
		}
	}
	return false
}