package config

import (
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/net/publicsuffix"
)

// corsPolicy decides which browser origins may call the API with credentials.
type corsPolicy struct {
	allowAll bool
	allowed  map[string]bool
	log      *logrus.Logger
	warned   sync.Map
}

func normalizeOrigin(origin string) string {
	return strings.ToLower(strings.TrimRight(strings.TrimSpace(origin), "/"))
}

func newCORSPolicy(config *viper.Viper, log *logrus.Logger) *corsPolicy {
	p := &corsPolicy{allowed: map[string]bool{}, log: log}
	// CORS_ALLOW_ALL=true restores the old "any origin" behaviour (not recommended).
	p.allowAll = strings.EqualFold(strings.TrimSpace(config.GetString("CORS_ALLOW_ALL")), "true")

	list := config.GetString("FRONTEND_URL") + "," + config.GetString("CORS_ALLOWED_ORIGINS")
	for _, origin := range strings.Split(list, ",") {
		if origin = normalizeOrigin(origin); origin != "" {
			p.allowed[origin] = true
		}
	}
	return p
}

func hostnameOf(hostport string) string {
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		return strings.ToLower(h)
	}
	return strings.ToLower(hostport)
}

// sameSite reports whether two hostnames belong to the same site: identical hosts, or
// names under the same registrable domain (app.spil.co.id and api.spil.co.id).
func sameSite(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	if a == b {
		return true
	}
	if net.ParseIP(a) != nil || net.ParseIP(b) != nil {
		return false
	}
	siteA, errA := publicsuffix.EffectiveTLDPlusOne(a)
	siteB, errB := publicsuffix.EffectiveTLDPlusOne(b)
	return errA == nil && errB == nil && siteA == siteB
}

func (p *corsPolicy) allows(origin string, r *http.Request) bool {
	if p.allowAll || p.allowed[normalizeOrigin(origin)] {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil || u.Hostname() == "" {
		return false
	}
	originHost := strings.ToLower(u.Hostname())

	// Behind a reverse proxy the public host is in X-Forwarded-Host.
	hosts := []string{hostnameOf(r.Host)}
	for _, h := range strings.Split(r.Header.Get("X-Forwarded-Host"), ",") {
		if h = strings.TrimSpace(h); h != "" {
			hosts = append(hosts, hostnameOf(h))
		}
	}
	for _, host := range hosts {
		if sameSite(originHost, host) {
			return true
		}
	}

	// Say once per origin how to allow it, so a misconfigured deployment is easy to diagnose.
	if _, seen := p.warned.LoadOrStore(origin, true); !seen && p.log != nil {
		p.log.Warnf("CORS: origin %q ditolak. Jika ini alamat frontend, isi FRONTEND_URL (atau CORS_ALLOWED_ORIGINS) dengan nilai tersebut.", origin)
	}
	return false
}
