// Package normalize provides the V2 request canonicalization stage.
package normalize

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"unicode/utf8"

	"zhiyuwaf/internal/core"
)

const (
	defaultMaxBodyBytes = 2 << 20
	maxDecodeRounds     = 3
)

// Normalizer converts transport input to consistent values before rules inspect
// it. It performs no blocking and never calls external systems.
type Normalizer struct {
	MaxBodyBytes int
}

func (Normalizer) Name() string { return "normalizer" }

func (n Normalizer) Process(ctx core.Context, req *core.RequestContext, state *core.PipelineState) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if req == nil {
		return fmt.Errorf("request is required")
	}
	maxBody := n.MaxBodyBytes
	if maxBody <= 0 {
		maxBody = defaultMaxBodyBytes
	}
	if len(req.Body) > maxBody {
		return fmt.Errorf("request body exceeds %d bytes", maxBody)
	}
	if req.Metadata == nil {
		req.Metadata = make(map[string]any)
	}
	if state.Metadata == nil {
		state.Metadata = make(map[string]any)
	}

	req.Method = strings.ToUpper(strings.TrimSpace(req.Method))
	req.Host = strings.ToLower(strings.TrimSpace(req.Host))
	req.Path = normalizePath(req.Path)
	req.Query = normalizeValues(req.Query)
	req.Header = normalizeHeaders(req.Header)

	contentType, _, _ := mime.ParseMediaType(req.Header.Get("Content-Type"))
	bodyText := normalizeText(string(req.Body))
	req.Metadata["normalized_body"] = bodyText
	req.Metadata["content_type"] = contentType
	req.Metadata["normalized_path"] = req.Path
	req.Metadata["normalized_query"] = canonicalValues(req.Query)
	req.Metadata["normalized_headers"] = canonicalHeaders(req.Header)

	switch {
	case strings.EqualFold(contentType, "application/json"):
		req.Metadata["normalized_json"] = normalizeJSON(req.Body)
	case strings.EqualFold(contentType, "application/x-www-form-urlencoded"):
		if form, err := url.ParseQuery(string(req.Body)); err == nil {
			req.Metadata["normalized_form"] = canonicalValues(normalizeValues(form))
		}
	case strings.HasPrefix(strings.ToLower(contentType), "multipart/form-data"):
		// Multipart boundaries may carry binary content. Preserve a text-only,
		// bounded representation for rules while avoiding file-body expansion.
		req.Metadata["normalized_multipart"] = bodyText
	}
	return nil
}

func normalizePath(raw string) string {
	if raw == "" {
		return "/"
	}
	decoded := repeatedDecode(raw)
	clean := path.Clean("/" + strings.TrimPrefix(decoded, "/"))
	if clean == "." {
		return "/"
	}
	return clean
}

func normalizeValues(values url.Values) url.Values {
	out := make(url.Values, len(values))
	for key, items := range values {
		key = normalizeText(key)
		for _, value := range items {
			out[key] = append(out[key], normalizeText(value))
		}
		sort.Strings(out[key])
	}
	return out
}

func normalizeHeaders(headers http.Header) http.Header {
	out := make(http.Header, len(headers))
	for key, items := range headers {
		canonicalKey := http.CanonicalHeaderKey(strings.TrimSpace(key))
		for _, value := range items {
			out[canonicalKey] = append(out[canonicalKey], normalizeText(value))
		}
		sort.Strings(out[canonicalKey])
	}
	return out
}

func normalizeText(raw string) string {
	value := strings.TrimSpace(raw)
	value = repeatedDecode(value)
	if !utf8.ValidString(value) {
		value = strings.ToValidUTF8(value, "�")
	}
	return strings.ToLower(value)
}

func repeatedDecode(raw string) string {
	value := raw
	for i := 0; i < maxDecodeRounds; i++ {
		decoded, err := url.QueryUnescape(value)
		if err != nil || decoded == value {
			break
		}
		value = decoded
	}
	return value
}

func canonicalValues(values url.Values) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+strings.Join(values[key], ","))
	}
	return strings.Join(parts, "&")
}

func canonicalHeaders(headers http.Header) string {
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, strings.ToLower(key))
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+":"+strings.Join(headers.Values(key), ","))
	}
	return strings.Join(parts, "\n")
}

func normalizeJSON(body []byte) string {
	if len(bytes.TrimSpace(body)) == 0 {
		return ""
	}
	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		return normalizeText(string(body))
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return normalizeText(string(body))
	}
	return normalizeText(string(canonical))
}

// NewContext is a small helper for consumers that do not otherwise need the
// standard context package in normalizer-only tests.
func NewContext() context.Context { return context.Background() }
