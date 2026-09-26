package httpapi

import "net/http"

const healthResponse = `{"status":"ok","service":"catalog-service"}`

// NewHandler creates the HTTP API for the Catalog Service skeleton.
func NewHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", health)
	return mux
}

func health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(healthResponse))
}
