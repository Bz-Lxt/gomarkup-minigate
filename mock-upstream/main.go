package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

func beijingNow() string {
	loc := time.FixedZone("CST", 8*3600)
	return time.Now().In(loc).Format("2006-01-02 15:04:05")
}

func main() {
	name := os.Getenv("INSTANCE_NAME")
	if name == "" {
		name = "unknown"
	}
	addr := os.Getenv("LISTEN")
	if addr == "" {
		addr = ":9001"
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"instance": name,
			"method":   r.Method,
			"path":     r.URL.Path,
			"query":    r.URL.RawQuery,
			"time":     beijingNow(),
			"headers": map[string]string{
				"X-Minigate-Route":    r.Header.Get("X-Minigate-Route"),
				"X-Minigate-Upstream": r.Header.Get("X-Minigate-Upstream"),
			},
		})
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"up"}`))
	})
	log.Printf("upstream %s listen %s", name, addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
