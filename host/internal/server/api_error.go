package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"regexp"
)

type apiErrorEnvelope struct {
	Error apiErrorPayload `json:"error"`
}

type apiErrorPayload struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Hint      string `json:"hint,omitempty"`
	RequestID string `json:"request_id"`
}

func writeAPIError(w http.ResponseWriter, status int, code string, message string, hint string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(apiErrorEnvelope{
		Error: apiErrorPayload{
			Code:      code,
			Message:   sanitizeErrText(message),
			Hint:      sanitizeErrText(hint),
			RequestID: newRequestID(),
		},
	})
}

func newRequestID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

var (
	reBearer = regexp.MustCompile(`(?i)\bAuthorization:\s*Bearer\s+[A-Za-z0-9._-]{6,}`)
	reTokenQ = regexp.MustCompile(`(?i)([?&](token|ticket)=)[^&\s"]+`)
	reJSONAT = regexp.MustCompile(`(?i)("?(access_token|refresh_token)"?\s*:\s*")[^"]+`)
)

func sanitizeErrText(s string) string {
	if s == "" {
		return ""
	}
	out := reBearer.ReplaceAllString(s, "Authorization: Bearer REDACTED")
	out = reTokenQ.ReplaceAllString(out, "$1REDACTED")
	out = reJSONAT.ReplaceAllString(out, `$1REDACTED`)
	return out
}

