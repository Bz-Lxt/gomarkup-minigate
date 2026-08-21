package health

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

func JoinHealthURL(target, path string) string {
	target = strings.TrimRight(target, "/")
	if path == "" {
		path = "/health"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return target + path
}

func Probe(client *http.Client, target, path string, expected int) error {
	if client == nil {
		client = http.DefaultClient
	}
	if expected <= 0 {
		expected = http.StatusOK
	}
	url := JoinHealthURL(target, path)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build probe %s: %w", url, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("probe %s: %w", url, err)
	}
	if expected != http.StatusOK {
		if resp.StatusCode != expected {
			return fmt.Errorf("probe %s status %d want %d", url, resp.StatusCode, expected)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		return nil
	}
	if resp.StatusCode >= 500 {
		return fmt.Errorf("probe %s status %d", url, resp.StatusCode)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	return nil
}

func Acceptable(status, expected int) bool {
	if expected > 0 {
		return status == expected
	}
	return status > 0 && status < 500
}
