package admin_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"minigate/internal/admin"
	"minigate/internal/config"
	"minigate/internal/model"
	"minigate/internal/router"
)

type acceptingSource struct{}

func (acceptingSource) Load() (*model.GatewayConfig, error) { return nil, errors.New("unused") }
func (acceptingSource) Save(*model.GatewayConfig) error     { return nil }
func (acceptingSource) Name() string                        { return "test" }

func TestUpdateRouteReportsApplyFailure(t *testing.T) {
	table := router.NewTable()
	table.Swap(&model.GatewayConfig{
		Upstreams: []model.UpstreamSpec{{ID: "users"}},
		Routes: []model.RouteSpec{{
			ID: "users-route", Name: "before", Path: "/users", Methods: []string{"GET"},
			UpstreamID: "users", Enabled: true,
		}},
	})
	reloader := config.NewReloader(acceptingSource{}, func(*model.GatewayConfig) error {
		return errors.New("runtime rejected configuration")
	})
	handler := (&admin.Server{Table: table, Reloader: reloader}).Handler()

	body := []byte(`{"name":"after","path":"/users","methods":["GET"],"upstream_id":"users","enabled":true}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/routes/users-route", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)

	if response.Code < 400 {
		t.Fatalf("update returned HTTP %d after runtime apply failed; body=%s", response.Code, response.Body.String())
	}

	get := httptest.NewRecorder()
	handler.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/api/v1/routes/users-route", nil))
	var result struct {
		Data struct {
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(get.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode route response: %v", err)
	}
	if result.Data.Name != "before" {
		t.Fatalf("runtime route changed despite rejected apply: name=%q", result.Data.Name)
	}
}
