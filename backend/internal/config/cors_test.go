package config

import (
	"net/http/httptest"
	"testing"

	"github.com/spf13/viper"
)

func policy(frontend, extra string) *corsPolicy {
	v := viper.New()
	v.Set("FRONTEND_URL", frontend)
	v.Set("CORS_ALLOWED_ORIGINS", extra)
	return newCORSPolicy(v, nil)
}

func TestCORSPolicy(t *testing.T) {
	cases := []struct {
		name, frontend, extra, origin, host, fwdHost string
		want                                         bool
	}{
		{"dev: same host, other port", "", "", "http://localhost:3000", "localhost:8080", "", true},
		{"server reached by IP", "", "", "http://10.0.0.5:3000", "10.0.0.5:8080", "", true},
		{"sub-domains of one site", "", "", "https://app.spil.co.id", "api.spil.co.id", "", true},
		{"same site via reverse proxy", "", "", "https://tds.spil.co.id", "tds_backend:8080", "api.spil.co.id", true},
		{"configured frontend on another site", "https://tds.example.com/", "", "https://tds.example.com", "api.internal:8080", "", true},
		{"extra allowed origin", "", "https://a.example.com, https://b.example.com", "https://b.example.com", "api.internal", "", true},
		{"foreign site is rejected", "https://app.spil.co.id", "", "https://evil.example.com", "api.spil.co.id", "", false},
		{"another company under .co.id is rejected", "", "", "https://app.other.co.id", "api.spil.co.id", "", false},
		{"different IP is rejected", "", "", "http://10.0.0.9:3000", "10.0.0.5:8080", "", false},
		{"garbage origin is rejected", "", "", "null", "api.spil.co.id", "", false},
	}
	for _, c := range cases {
		r := httptest.NewRequest("GET", "http://"+c.host+"/x", nil)
		r.Host = c.host
		if c.fwdHost != "" {
			r.Header.Set("X-Forwarded-Host", c.fwdHost)
		}
		if got := policy(c.frontend, c.extra).allows(c.origin, r); got != c.want {
			t.Errorf("%s: allows(%q) = %v, want %v", c.name, c.origin, got, c.want)
		}
	}
}

func TestCORSAllowAllEscapeHatch(t *testing.T) {
	v := viper.New()
	v.Set("CORS_ALLOW_ALL", "true")
	r := httptest.NewRequest("GET", "http://api.spil.co.id/x", nil)
	if !newCORSPolicy(v, nil).allows("https://evil.example.com", r) {
		t.Error("CORS_ALLOW_ALL=true must allow any origin")
	}
}
