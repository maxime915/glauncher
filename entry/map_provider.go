package entry

import (
	"bytes"
	"io"
	"strings"
)

type MapProvider[T Entry] struct {
	Content map[string]T
	Prefix  string
}

func (mp MapProvider[T]) GetEntryReader() (io.Reader, error) {
	buf := &bytes.Buffer{}

	writePrefix := mp.Prefix != ""

	for key := range mp.Content {
		// err of WriteXxx is always nil, can safely be ignored
		if writePrefix {
			buf.WriteString(mp.Prefix)
		}
		buf.WriteString(key)
		buf.WriteRune('\n')
	}

	return buf, nil
}

func (mp MapProvider[T]) Fetch(entry string) (Entry, bool) {
	// remove prefix
	if mp.Prefix != "" {
		entry = strings.TrimPrefix(entry, mp.Prefix)
	}

	value, ok := mp.Content[entry]
	return value, ok
}
