package locale

import (
	"net/http"
	"strings"
)

func GetPreferredLocale(r *http.Request) string {
	accept := r.Header.Get("Accept-Language")
	if accept == "" {
		return "en-US"
	}

	parts := strings.Split(accept, ",")
	if len(parts) == 0 {
		return "en-US"
	}

	locale := strings.TrimSpace(parts[0])
	if locale == "" {
		return "en-US"
	}

	return locale
}
