package util

import (
	"fmt"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mozillazg/go-pinyin"

	"ani-rss/internal/model"
)

// NewRequest creates a request with the standard UA.
func NewRequest(method, rawURL string) (*http.Request, error) {
	req, err := http.NewRequest(method, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent())
	return req, nil
}

// LogWarn logs a warning through the internal log package.
func LogWarn(logger, format string, args ...interface{}) {
	logWarnFn(logger, fmt.Sprintf(format, args...))
}

var logWarnFn = func(logger, msg string) {}

// SetLogWarn registers the internal log warning sink.
func SetLogWarn(fn func(logger, msg string)) { logWarnFn = fn }

var videoExts = map[string]bool{
	".mp4": true, ".mkv": true, ".avi": true, ".wmv": true, ".mov": true, ".ts": true, ".flv": true, ".rmvb": true, ".rm": true, ".webm": true,
}

var subtitleExts = map[string]bool{
	".ass": true, ".ssa": true, ".sub": true, ".srt": true, ".lyc": true, ".sup": true, ".pgs": true, ".mks": true,
}

var videoMimes = map[string]string{
	".mp4": "video/mp4", ".m4v": "video/x-m4v", ".mkv": "video/x-matroska",
	".avi": "video/x-msvideo", ".wmv": "video/x-ms-wmv", ".mov": "video/quicktime",
	".ts": "video/mp2t", ".flv": "video/x-flv", ".rmvb": "video/vnd.rn-realvideo",
	".rm": "video/vnd.rn-realvideo", ".webm": "video/webm",
}

// VideoMimeType returns the media type for a video path, or "" when unknown.
func VideoMimeType(name string) string {
	return videoMimes[strings.ToLower(filepath.Ext(name))]
}

// IsVideo reports whether the extension is a video file.
func IsVideo(name string) bool { return videoExts[strings.ToLower(filepath.Ext(name))] }

// IsSubtitle reports whether the extension is a subtitle file.
func IsSubtitle(name string) bool { return subtitleExts[strings.ToLower(filepath.Ext(name))] }

// FormatSize converts a byte count to a human-readable string like "1.23 MiB"
// (1024-based, units B/KiB/MiB/GiB/TiB, mirroring FileUtils.formatSize).
func FormatSize(size int64) string {
	if size <= 0 {
		return "0 B"
	}
	units := []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB"}
	i := 0
	f := float64(size)
	for f >= 1024 && i < len(units)-1 {
		f /= 1024
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%d B", size)
	}
	return fmt.Sprintf("%.2f %s", f, units[i])
}

var (
	episodeTokenRe = regexp.MustCompile(`[Ss](\d+)[Ee](\d+(\.5)?)`)
	pinyinArgs     = pinyin.Args{
		Style:     pinyin.Normal,
		Heteronym: false,
		Separator: "",
		Fallback:  func(r rune, a pinyin.Args) []string { return []string{string(r)} },
	}
)

// GetPinyin returns the pinyin of a Chinese string.
func GetPinyin(s string) string {
	var sb strings.Builder
	for _, r := range s {
		res := pinyin.Pinyin(string(r), pinyinArgs)
		if len(res) > 0 && len(res[0]) > 0 {
			sb.WriteString(res[0][0])
		} else {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// GetPinyinInitials returns the initials (first letter of each char's pinyin).
func GetPinyinInitials(s string) string {
	var sb strings.Builder
	for _, r := range s {
		res := pinyin.Pinyin(string(r), pinyinArgs)
		if len(res) > 0 && len(res[0]) > 0 {
			sb.WriteString(res[0][0][:1])
		} else {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// GetEpisodeToken extracts the "SxxExx" token from a name, or "".
func GetEpisodeToken(name string) string {
	m := episodeTokenRe.FindString(name)
	if m == "" {
		return ""
	}
	// canonicalize
	mm := episodeTokenRe.FindStringSubmatch(name)
	if len(mm) < 3 {
		return m
	}
	return fmt.Sprintf("S%sE%s", mm[1], mm[2])
}

var regHasChinese = regexp.MustCompile(`[\p{Han}]`)

// HasChinese reports whether a string contains Chinese characters.
func HasChinese(s string) bool { return regHasChinese.MatchString(s) }

// IsBlank reports whether a string is empty or whitespace.
func IsBlank(s string) bool { return strings.TrimSpace(s) == "" }

// FirstNonBlank returns the first non-blank line of s, or "".
func FirstNonBlank(s string) string {
	for _, line := range strings.Split(s, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			return line
		}
	}
	return ""
}

// URLBase returns the base URL (scheme+host) of a URL string.
func URLBase(raw string) string {
	idx := strings.Index(raw, "://")
	if idx < 0 {
		return raw
	}
	rest := raw[idx+3:]
	slash := strings.Index(rest, "/")
	if slash < 0 {
		return raw
	}
	return raw[:idx+3+slash]
}

var weekRegexes = []*regexp.Regexp{
	regexp.MustCompile(`(星期|周)日`),
	regexp.MustCompile(`(星期|周)一`),
	regexp.MustCompile(`(星期|周)二`),
	regexp.MustCompile(`(星期|周)三`),
	regexp.MustCompile(`(星期|周)四`),
	regexp.MustCompile(`(星期|周)五`),
	regexp.MustCompile(`(星期|周)六`),
}

// weekPatternIndex maps a weekday label to its pattern index (0=Sunday).
func weekPatternIndex(label string) int {
	for i, re := range weekRegexes {
		if re.MatchString(label) {
			return i
		}
	}
	return -1
}

// WeekSortIndex returns the sort position of a weekday label when ordered
// starting from today's weekday then wrapping (mirrors WeekComparator).
func WeekSortIndex(label string) int {
	idx := weekPatternIndex(label)
	if idx < 0 {
		return 1 << 30
	}
	today := int(time.Now().In(model.Loc()).Weekday()) // 0=Sunday
	var order []int
	for i := today; i >= 0; i-- {
		order = append(order, i)
	}
	for i := 6; i > 0; i-- {
		if !containsInt(order, i) {
			order = append(order, i)
		}
	}
	for i, v := range order {
		if v == idx {
			return i
		}
	}
	return 1 << 30
}

func containsInt(list []int, v int) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}