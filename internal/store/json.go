package store

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// ReadJSONFromBytes parses JSON bytes into v.
func ReadJSONFromBytes(b []byte, v interface{}) error {
	return json.Unmarshal(b, v)
}

// ReadJSON reads and parses a JSON file. If the file is missing it writes
// the default value to disk and returns it (matching JsonReader behavior).
func ReadJSON(path string, def interface{}) error {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return WriteJSON(path, def)
		}
		return err
	}
	if len(bytesTrimSpace(b)) == 0 {
		return WriteJSON(path, def)
	}
	return json.Unmarshal(b, def)
}

// WriteJSON writes the value as pretty JSON to a temp file then atomically
// moves it into place (matching JsonWriter behavior).
func WriteJSON(path string, v interface{}) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".temp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func bytesTrimSpace(b []byte) []byte {
	start, end := 0, len(b)
	for start < end && (b[start] == ' ' || b[start] == '\n' || b[start] == '\r' || b[start] == '\t') {
		start++
	}
	for end > start && (b[end-1] == ' ' || b[end-1] == '\n' || b[end-1] == '\r' || b[end-1] == '\t') {
		end--
	}
	return b[start:end]
}