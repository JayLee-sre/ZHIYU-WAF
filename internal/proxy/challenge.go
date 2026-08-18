package proxy

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var cookieSecret []byte

func init() {
	cookieSecret = make([]byte, 32)
	rand.Read(cookieSecret)
}

func signCookie(value string) string {
	mac := hmac.New(sha256.New, cookieSecret)
	mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}

func verifyCookie(cookie string) bool {
	parts := strings.SplitN(cookie, ".", 2)
	if len(parts) != 2 {
		return false
	}
	ts := parts[0]
	sig := parts[1]

	expectedSig := signCookie(ts)
	if !hmac.Equal([]byte(sig), []byte(expectedSig)) {
		return false
	}

	// Challenge cookie expires after 24 hours.
	var timestamp int64
	fmt.Sscanf(ts, "%d", &timestamp)
	return time.Now().Unix()-timestamp <= 86400
}

func makeVerifiedCookie() string {
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	return timestamp + "." + signCookie(timestamp)
}

func getCookieValue(req *http.Request, name string) string {
	for _, cookie := range req.Header["Cookie"] {
		for _, part := range strings.Split(cookie, ";") {
			keyValue := strings.SplitN(strings.TrimSpace(part), "=", 2)
			if len(keyValue) == 2 && keyValue[0] == name {
				return keyValue[1]
			}
		}
	}
	return ""
}

func isStaticAsset(path string) bool {
	for _, extension := range []string{".css", ".js", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico", ".woff", ".woff2", ".ttf", ".eot"} {
		if len(path) >= len(extension) && path[len(path)-len(extension):] == extension {
			return true
		}
	}
	return false
}

// challengeHTML is an embedded, standalone verification page. Keeping the UI
// in a dedicated HTML file makes the public-facing challenge experience easier
// to review and evolve without changing the verification contract.
//
//go:embed assets/challenge.html
var challengeHTML string
