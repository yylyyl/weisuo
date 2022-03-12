package serverhelper

import (
	"net/http"
	"strings"
)

func DefaultRealIpFunc(r *http.Request) string {
	// TODO: IPv6
	colonPos := strings.Index(r.RemoteAddr, ":")
	return r.RemoteAddr[:colonPos]
}
