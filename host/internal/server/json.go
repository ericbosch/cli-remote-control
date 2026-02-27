package server

import (
	"encoding/json"
	"net/http"
)

func jsonDecode(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}
