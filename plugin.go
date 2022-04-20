package geoblock

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/mdouchement/geoblock/lookup"
)

// A Plugin is the struct used by Traefik to execute custom actions.
type Plugin struct {
	Config
	name      string
	next      http.Handler
	evaluator *Evaluator
}

// New creates a new plugin instance.
func New(_ context.Context, next http.Handler, c *Config, name string) (http.Handler, error) {
	if next == nil {
		return nil, fmt.Errorf("%s: no next handler provided", name)
	}

	if c == nil {
		return nil, fmt.Errorf("%s: no config provided", name)
	}

	if c.DefaultAction != DefaultActionAllow && c.DefaultAction != DefaultActionBlock {
		return nil, fmt.Errorf("%s: invalid default action: %s", name, c.DefaultAction)
	}

	p := &Plugin{
		Config: *c,
		name:   name,
		next:   next,
	}

	if !c.Enabled {
		log.Printf("%s: disabled", name)
		return p, nil
	}

	if http.StatusText(c.DisallowedStatusCode) == "" {
		return nil, fmt.Errorf("%s: %d is not a valid http status code", name, c.DisallowedStatusCode)
	}

	if len(c.Databases) == 0 {
		return nil, fmt.Errorf("%s: no database file path configured", name)
	}

	//

	var err error

	p.evaluator, err = NewEvaluator(name, *c)
	if err != nil {
		return nil, fmt.Errorf("%s: evaluator: %w", name, err)
	}

	for _, databasename := range c.Databases {
		lookup, err := lookup.OpenIP2location(databasename)
		if err != nil {
			return nil, fmt.Errorf("%s: %s: ip2location: %w", name, databasename, err)
		}

		p.evaluator.AddLookup(lookup)
	}

	return p, err
}

// ServeHTTP implements the http.Handler interface.
func (p Plugin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !p.Enabled {
		p.next.ServeHTTP(w, r)
		return
	}

	for _, ip := range p.CollectIPs(r) {
		allowed, country, err := p.evaluator.Evaluate(ip)
		if err != nil {
			log.Printf("%s: [%s %s %s] - %v", p.name, r.Host, r.Method, r.URL.Path, err)
			w.WriteHeader(p.DisallowedStatusCode)
			return
		}

		if !allowed {
			log.Printf("%s: [%s %s %s] blocked request from %s", p.name, r.Host, r.Method, r.URL.Path, strings.ToUpper(country))
			w.WriteHeader(p.DisallowedStatusCode)
			return
		}
	}

	p.next.ServeHTTP(w, r)
}

// CollectIPs collects the remote IPs from the X-Forwarded-For and X-Real-IP headers.
func (p Plugin) CollectIPs(r *http.Request) []string {
	m := make(map[string]bool)

	if ips := r.Header.Get("X-Forwarded-For"); ips != "" {
		for _, ip := range strings.Split(ips, ",") {
			ip = strings.TrimSpace(ip)
			if ip == "" {
				continue
			}

			m[ip] = true
		}
	}

	if ips := r.Header.Get("X-Real-IP"); ips != "" {
		for _, ip := range strings.Split(ips, ",") {
			ip = strings.TrimSpace(ip)
			if ip == "" {
				continue
			}

			m[ip] = true
		}
	}

	var ips []string
	for k := range m {
		ips = append(ips, k)
	}

	return ips
}
