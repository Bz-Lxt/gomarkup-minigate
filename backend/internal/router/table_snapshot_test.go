package router_test

import (
	"testing"

	"minigate/internal/model"
	"minigate/internal/router"
)

func TestTableSnapshotDoesNotShareRouteStorage(t *testing.T) {
	cfg := &model.GatewayConfig{
		Routes: []model.RouteSpec{{
			ID:         "orders",
			Name:       "orders",
			Path:       "/orders",
			Methods:    []string{"GET"},
			UpstreamID: "orders-api",
			Enabled:    true,
		}},
	}
	table := router.NewTable()
	table.Swap(cfg)

	routes := table.Routes()
	routes[0] = model.RouteSpec{
		ID:         "payments",
		Name:       "payments",
		Path:       "/payments",
		Methods:    []string{"POST"},
		UpstreamID: "payments-api",
		Enabled:    true,
	}

	fresh := table.Routes()
	if len(fresh) != 1 || fresh[0].ID != "orders" || fresh[0].Path != "/orders" {
		t.Fatalf("active route changed after editing returned routes: %+v", fresh)
	}
	if route, _ := table.Match("GET", "/orders", ""); route == nil {
		t.Fatalf("applied route no longer matches: %+v", route)
	}
	if route, _ := table.Match("POST", "/payments", ""); route != nil {
		t.Fatalf("unapplied route unexpectedly matches: %+v", route)
	}
}
