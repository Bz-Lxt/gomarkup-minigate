package model

type NodeSpec struct {
	Target string `json:"target" yaml:"target"`
	Weight int    `json:"weight" yaml:"weight"`
}

type CircuitSpec struct {
	Enabled          bool `json:"enabled" yaml:"enabled"`
	FailureThreshold int  `json:"failure_threshold" yaml:"failure_threshold"`
	SuccessThreshold int  `json:"success_threshold" yaml:"success_threshold"`
	OpenTimeoutMS    int  `json:"open_timeout_ms" yaml:"open_timeout_ms"`
}

type UpstreamSpec struct {
	ID              string      `json:"id" yaml:"id"`
	Name            string      `json:"name" yaml:"name"`
	Algorithm       string      `json:"algorithm" yaml:"algorithm"`
	TimeoutMS       int         `json:"timeout_ms" yaml:"timeout_ms"`
	FailThreshold   int         `json:"fail_threshold" yaml:"fail_threshold"`
	HealthPath      string      `json:"health_path,omitempty" yaml:"health_path,omitempty"`
	ExpectedStatus  int         `json:"expected_status,omitempty" yaml:"expected_status,omitempty"`
	ProbeIntervalMS int         `json:"probe_interval_ms,omitempty" yaml:"probe_interval_ms,omitempty"`
	Circuit         CircuitSpec `json:"circuit" yaml:"circuit"`
	Nodes           []NodeSpec  `json:"nodes" yaml:"nodes"`
}

type MiddlewareSpec struct {
	Name    string         `json:"name" yaml:"name"`
	Enabled bool           `json:"enabled" yaml:"enabled"`
	Config  map[string]any `json:"config" yaml:"config"`
}

type RouteSpec struct {
	ID          string   `json:"id" yaml:"id"`
	Name        string   `json:"name" yaml:"name"`
	Path        string   `json:"path" yaml:"path"`
	Methods     []string `json:"methods" yaml:"methods"`
	Host        string   `json:"host" yaml:"host"`
	UpstreamID  string   `json:"upstream_id" yaml:"upstream_id"`
	Middlewares []string `json:"middlewares" yaml:"middlewares"`
	Enabled     bool     `json:"enabled" yaml:"enabled"`
	Priority    int      `json:"priority" yaml:"priority"`
	StripPrefix string   `json:"strip_prefix" yaml:"strip_prefix"`
	CreatedAt   string   `json:"created_at,omitempty" yaml:"created_at,omitempty"`
	UpdatedAt   string   `json:"updated_at,omitempty" yaml:"updated_at,omitempty"`
}

type GatewayConfig struct {
	Listen            string           `json:"listen" yaml:"listen"`
	AdminListen       string           `json:"admin_listen" yaml:"admin_listen"`
	LogLevel          string           `json:"log_level" yaml:"log_level"`
	GlobalMiddlewares []MiddlewareSpec `json:"global_middlewares" yaml:"global_middlewares"`
	Upstreams         []UpstreamSpec   `json:"upstreams" yaml:"upstreams"`
	Routes            []RouteSpec      `json:"routes" yaml:"routes"`
}

type NodeRuntime struct {
	Target  string `json:"target"`
	Weight  int    `json:"weight"`
	Healthy bool   `json:"healthy"`
	Fails   int32  `json:"fails"`
}

type UpstreamStatus struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Algorithm string        `json:"algorithm"`
	Healthy   int           `json:"healthy"`
	Total     int           `json:"total"`
	Nodes     []NodeRuntime `json:"nodes,omitempty"`
}

type ErrorEvent struct {
	Time    string `json:"time"`
	Message string `json:"message"`
}

type HotReloadStatus struct {
	Source      string `json:"source"`
	LastSuccess string `json:"last_success"`
	LastError   string `json:"last_error"`
}

type Stats struct {
	QPS           float64           `json:"qps"`
	TotalRequests uint64            `json:"total_requests"`
	ActiveRoutes  int               `json:"active_routes"`
	Upstreams     []UpstreamStatus  `json:"upstreams"`
	RecentErrors  []ErrorEvent      `json:"recent_errors"`
	HotReload     HotReloadStatus   `json:"hot_reload"`
	Circuits      map[string]string `json:"circuits,omitempty"`
}
