package matroska

import (
	"errors"
	"io"
	"os"
	"strings"
)

// EBML element IDs.
const (
	idEBML        = 0x1A45DFA3
	idSegment     = 0x18538067
	idInfo        = 0x1549A966
	idTracks      = 0x1654AE6B
	idTrackEntry  = 0xAE
	idCluster     = 0x1F43B675
	idTimestamp   = 0xE7
	idSimpleBlock = 0xA3
	idBlockGroup  = 0xA0
	idBlock       = 0xA1
	idTrackType   = 0x83
	idCodecID     = 0x86
	idName        = 0x536E
	idLanguage    = 0x22B59C
	idTrackNumber = 0xD7
	idVoid        = 0xEC
)

const (
	trackVideo      = 1
	trackAudio      = 2
	trackSubtitle   = 0x11
)

// Subtitle is one extracted subtitle track.
type Subtitle struct {
	Name     string
	Language string
	Codec    string
	Content  string // VTT content
}

// reader streams an EBML element tree from an io.ReaderAt. Elements are read
// header-only; payloads are skipped with a seek, so only subtitle block data is
// ever buffered regardless of file size.
type reader struct {
	ra   io.ReaderAt
	size int64
	pos  int64
}

func (r *reader) done() bool { return r.pos >= r.size }

func (r *reader) skip(n int64) {
	r.pos += n
	if r.pos > r.size {
		r.pos = r.size
	}
}

func (r *reader) readAt(n int64) ([]byte, bool) {
	if n == 0 {
		return []byte{}, true
	}
	if n < 0 || r.pos+n > r.size {
		return nil, false
	}
	b := make([]byte, n)
	if _, err := r.ra.ReadAt(b, r.pos); err != nil {
		return nil, false
	}
	r.pos += n
	return b, true
}

// readVint reads an EBML VINT: first byte carries the length marker.
// Returns the value (id keeps marker bit, length clears it) and the vint size.
func (r *reader) readVint(clearMarker bool) (uint64, int, error) {
	hdr, ok := r.readAt(1)
	if !ok {
		return 0, 0, errors.New("EOF")
	}
	first := hdr[0]
	var mask byte
	var length int
	for i := 0; i < 8; i++ {
		bit := byte(1 << (7 - i))
		if first&bit != 0 {
			mask = bit
			length = i + 1
			break
		}
	}
	if length == 0 {
		return 0, 0, errors.New("bad vint")
	}
	rest, ok := r.readAt(int64(length) - 1)
	if !ok {
		return 0, 0, errors.New("EOF")
	}
	val := uint64(first)
	for _, c := range rest {
		val = val<<8 | uint64(c)
	}
	if clearMarker {
		val &^= uint64(mask) << (8 * (length - 1))
	}
	return val, length, nil
}

func (r *reader) readElement() (id uint64, size uint64, err error) {
	id, _, err = r.readVint(false)
	if err != nil {
		return
	}
	size, _, err = r.readVint(true)
	return
}

func (r *reader) readUint(size uint64) uint64 {
	b, ok := r.readAt(int64(size))
	if !ok {
		return 0
	}
	var v uint64
	for _, c := range b {
		v = v<<8 | uint64(c)
	}
	return v
}

func (r *reader) readBytes(n uint64) []byte {
	b, _ := r.readAt(int64(n))
	return b
}

func (r *reader) readInt16() int16 {
	b, ok := r.readAt(2)
	if !ok {
		return 0
	}
	return int16(b[0])<<8 | int16(b[1])
}

type subtitleTrack struct {
	number   uint64
	codec    string
	name     string
	language string
}

type cue struct {
	start int64
	text  string
}

// ExtractSubtitles parses an MKV file and returns embedded subtitle tracks
// converted to VTT. The file is read incrementally: element headers are read
// as needed and non-subtitle block payloads are skipped, so memory use stays
// proportional to the subtitle data rather than the whole file.
func ExtractSubtitles(path string) ([]Subtitle, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	r := &reader{ra: f, size: fi.Size()}

	// EBML header
	if err := skipTop(r, idEBML); err != nil {
		return nil, err
	}
	// Segment header: read its id/size but do NOT skip the payload — the
	// Tracks and Clusters we need live inside the segment.
	segID, _, err := r.readElement()
	if err != nil {
		return nil, err
	}
	if segID != idSegment {
		return nil, errors.New("unexpected top element")
	}

	// first pass: find the Tracks element
	var tracks []subtitleTrack
	segStart := r.pos
	for !r.done() {
		id, size, err := r.readElement()
		if err != nil {
			break
		}
		if id == idTracks {
			tracks = parseTracks(r, size)
		} else {
			r.skip(int64(size))
		}
		if len(tracks) > 0 {
			break
		}
	}

	if len(tracks) == 0 {
		return nil, nil
	}

	// second pass: collect subtitle cues by scanning clusters
	byNumber := map[uint64][]cue{}
	r.pos = segStart
	for !r.done() {
		id, size, err := r.readElement()
		if err != nil {
			break
		}
		if id == idCluster {
			collectCluster(r, size, tracks, byNumber)
		} else {
			r.skip(int64(size))
		}
	}

	var subs []Subtitle
	for _, t := range tracks {
		cues := byNumber[t.number]
		if len(cues) == 0 {
			continue
		}
		subs = append(subs, Subtitle{
			Name:     t.name,
			Language: t.language,
			Codec:    t.codec,
			Content:  buildVTT(cues),
		})
	}
	return subs, nil
}

func skipTop(r *reader, wantID uint64) error {
	id, size, err := r.readElement()
	if err != nil {
		return err
	}
	if id != wantID {
		return errors.New("unexpected top element")
	}
	r.skip(int64(size))
	return nil
}

// parseTracks iterates the Tracks element bounded by size.
func parseTracks(r *reader, size uint64) []subtitleTrack {
	var tracks []subtitleTrack
	end := r.pos + int64(size)
	if end > r.size {
		end = r.size
	}
	for r.pos < end {
		id, esize, err := r.readElement()
		if err != nil || esize == 0 {
			break
		}
		if id == idTrackEntry {
			if track := parseTrackEntry(r, esize); track != nil {
				tracks = append(tracks, *track)
			}
		} else {
			r.skip(int64(esize))
		}
	}
	return tracks
}

// parseTrackEntry parses one TrackEntry bounded by size.
func parseTrackEntry(r *reader, size uint64) *subtitleTrack {
	track := &subtitleTrack{}
	end := r.pos + int64(size)
	if end > r.size {
		end = r.size
	}
	for r.pos < end {
		id, esize, err := r.readElement()
		if err != nil || esize == 0 {
			break
		}
		switch id {
		case idTrackNumber:
			track.number = r.readUint(esize)
		case idTrackType:
			// read and ignore (only subtitle tracks are kept below)
			r.skip(int64(esize))
		case idCodecID:
			track.codec = string(r.readBytes(esize))
		case idName:
			track.name = string(r.readBytes(esize))
		case idLanguage:
			track.language = string(r.readBytes(esize))
		default:
			r.skip(int64(esize))
		}
	}
	if track.number > 0 && (track.codec == "S_TEXT/UTF8" || track.codec == "S_TEXT/ASS") {
		return track
	}
	return nil
}

func collectCluster(r *reader, size uint64, tracks []subtitleTrack, byNumber map[uint64][]cue) {
	subNums := map[uint64]subtitleTrack{}
	for _, t := range tracks {
		subNums[t.number] = t
	}
	end := r.pos + int64(size)
	if end > r.size {
		end = r.size
	}
	var clusterTime int64
	for r.pos < end {
		id, esize, err := r.readElement()
		if err != nil || esize == 0 {
			break
		}
		switch id {
		case idTimestamp:
			clusterTime = int64(r.readUint(esize))
		case idSimpleBlock:
			parseBlock(r, esize, clusterTime, subNums, byNumber)
		case idBlockGroup:
			parseBlockGroup(r, esize, clusterTime, subNums, byNumber)
		default:
			r.skip(int64(esize))
		}
	}
}

func parseBlockGroup(r *reader, size uint64, clusterTime int64, subNums map[uint64]subtitleTrack, byNumber map[uint64][]cue) {
	end := r.pos + int64(size)
	if end > r.size {
		end = r.size
	}
	for r.pos < end {
		id, esize, err := r.readElement()
		if err != nil || esize == 0 {
			break
		}
		if id == idBlock {
			parseBlock(r, esize, clusterTime, subNums, byNumber)
		} else {
			r.skip(int64(esize))
		}
	}
}

// parseBlock reads a block header, then buffers the payload only when the block
// belongs to a subtitle track (video/audio payloads are skipped).
func parseBlock(r *reader, size uint64, clusterTime int64, subNums map[uint64]subtitleTrack, byNumber map[uint64][]cue) {
	end := r.pos + int64(size)
	if end > r.size {
		end = r.size
	}
	trackNum, _, err := r.readVint(true)
	if err != nil {
		r.pos = end
		return
	}
	timecode := r.readInt16()
	// flags byte (lacing); assume none
	r.skip(1)

	track, isSub := subNums[trackNum]
	payloadStart := r.pos
	r.pos = end
	if !isSub {
		return
	}
	r.pos = payloadStart
	data := r.readBytes(uint64(end - payloadStart))
	text := decodeSubtitleText(track.codec, data)
	if text != "" {
		start := clusterTime + int64(timecode)
		byNumber[trackNum] = append(byNumber[trackNum], cue{start: start, text: text})
	}
}

func decodeSubtitleText(codec string, data []byte) string {
	s := strings.TrimSpace(string(data))
	if s == "" {
		return ""
	}
	if strings.HasPrefix(codec, "S_TEXT/UTF8") {
		return s
	}
	if strings.HasPrefix(codec, "S_TEXT/ASS") {
		// SSA Dialogue: 0:0:0:0,,text,...  -> take after the 9th comma
		if strings.HasPrefix(s, "Dialogue:") {
			s = strings.TrimPrefix(s, "Dialogue:")
		}
		parts := strings.SplitN(s, ",", 10)
		if len(parts) >= 10 {
			return strings.TrimSpace(parts[9])
		}
		// fallback: strip {...} override tags
		return stripAssOverrides(strings.TrimSpace(s))
	}
	return ""
}

func stripAssOverrides(s string) string {
	var sb strings.Builder
	inTag := false
	for _, c := range s {
		if c == '{' {
			inTag = true
			continue
		}
		if c == '}' {
			inTag = false
			continue
		}
		if !inTag {
			sb.WriteRune(c)
		}
	}
	return strings.TrimSpace(sb.String())
}

func buildVTT(cues []cue) string {
	var sb strings.Builder
	sb.WriteString("WEBVTT\n\n")
	n := 1
	for i := 0; i < len(cues); i++ {
		c := cues[i]
		end := c.start + 3000
		if i+1 < len(cues) && cues[i+1].start > c.start {
			end = cues[i+1].start
		}
		text := c.text
		text = strings.ReplaceAll(text, "\n", "\n")
		sb.WriteString(itoa(int64(n)) + "\n")
		sb.WriteString(formatVttTime(c.start) + " --> " + formatVttTime(end) + "\n")
		sb.WriteString(text + "\n\n")
		n++
	}
	return sb.String()
}

func formatVttTime(ms int64) string {
	ms = ms % (24 * 3600 * 1000)
	h := ms / 3600000
	m := (ms / 60000) % 60
	s := (ms / 1000) % 60
	mm := ms % 1000
	return pad2(h) + ":" + pad2(m) + ":" + pad2(s) + "." + pad3(mm)
}

func pad2(n int64) string {
	if n < 10 {
		return "0" + itoa(n)
	}
	return itoa(n)
}

func pad3(n int64) string {
	if n < 10 {
		return "00" + itoa(n)
	}
	if n < 100 {
		return "0" + itoa(n)
	}
	return itoa(n)
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}