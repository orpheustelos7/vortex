package config

type Policy struct {
	TenantID   string   `json:"tenant_id"`
	Rate       float64  `json:"rate"`
	Burst      int64    `json:"burst"`
	BackendURL string   `json:"backend_url"`
	APIKeys    []string `json:"api_keys,omitempty"`
	JWTSecret  string   `json:"jwt_secret,omitempty"`
}

func (p Policy) Valid() bool {
	return p.TenantID != "" && p.Rate > 0 && p.Burst > 0 && p.BackendURL != ""
}
