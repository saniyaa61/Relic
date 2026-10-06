package importer

import (
	"encoding/base64"
	"errors"
	"net/url"
	"strings"

	"github.com/saniyaa61/relic/store"
)

// resizePoster decodes a data: URL image and resizes it for storage
// (store.ResizePoster).
func resizePoster(dataURL string) ([]byte, error) {
	raw, err := decodeDataURL(dataURL)
	if err != nil {
		return nil, err
	}
	return store.ResizePoster(raw)
}

func decodeDataURL(s string) ([]byte, error) {
	if !strings.HasPrefix(s, "data:") {
		return nil, errors.New("not an embedded image")
	}
	meta, data, ok := strings.Cut(s[len("data:"):], ",")
	if !ok {
		return nil, errors.New("malformed data URL")
	}
	if strings.HasSuffix(meta, ";base64") {
		return base64.StdEncoding.DecodeString(strings.TrimSpace(data))
	}
	d, err := url.PathUnescape(data)
	return []byte(d), err
}
