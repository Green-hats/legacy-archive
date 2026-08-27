package model

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

var loc = func() *time.Location {
	l, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("GMT+8", 8*3600)
	}
	return l
}()

// Loc returns the GMT+8 location used for all date formatting.
func Loc() *time.Location { return loc }

// Now returns the current time in the configured location.
func Now() time.Time { return time.Now().In(loc) }

// DateTime serializes as yyyy-MM-dd HH:mm:ss (matching GsonStatic date format).
type DateTime time.Time

func (d DateTime) Time() time.Time { return time.Time(d) }

func (d *DateTime) MarshalJSON() ([]byte, error) {
	if d == nil {
		return []byte("null"), nil
	}
	return []byte(`"` + time.Time(*d).Format("2006-01-02 15:04:05") + `"`), nil
}

func (d *DateTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "null" {
		*d = DateTime(time.Time{})
		return nil
	}
	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02", time.RFC3339} {
		if t, err := time.ParseInLocation(layout, s, loc); err == nil {
			*d = DateTime(t)
			return nil
		}
	}
	return nil
}

// Date serializes as yyyy-MM-dd (matching DateAdapter).
type Date time.Time

func (d Date) Time() time.Time { return time.Time(d) }

func (d *Date) MarshalJSON() ([]byte, error) {
	if d == nil {
		return []byte("null"), nil
	}
	t := time.Time(*d)
	if t.IsZero() {
		return []byte(`""`), nil
	}
	return []byte(`"` + t.Format("2006-01-02") + `"`), nil
}

func (d *Date) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "null" {
		*d = Date(time.Time{})
		return nil
	}
	for _, layout := range []string{"2006-01-02", "2006", "2006-01-02 15:04:05"} {
		if t, err := time.ParseInLocation(layout, s, loc); err == nil {
			*d = Date(t)
			return nil
		}
	}
	return nil
}

// StrID is a string ID that also accepts JSON numbers on decode
// (the animes.garden API returns numeric ids; Gson coerces them to strings).
type StrID string

func (s *StrID) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch t := v.(type) {
	case string:
		*s = StrID(t)
	case float64:
		*s = StrID(strconv.FormatFloat(t, 'f', -1, 64))
	case json.Number:
		*s = StrID(t.String())
	default:
		*s = ""
	}
	return nil
}