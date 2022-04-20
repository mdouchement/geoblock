package geoblock

import (
	"fmt"
	"net"
	"strings"

	"github.com/mdouchement/geoblock/lookup"
)

// An Evaluator evaluates whether an IP is allowed or blocked.
type Evaluator struct {
	name    string
	lookups []lookup.Lookup

	fallback       string
	allowedCIDR    []*net.IPNet
	allowedCountry map[string]bool
	blockedCIDR    []*net.IPNet
	blockedCountry map[string]bool
}

// NewEvaluator returns a new Evaluator.
func NewEvaluator(name string, c Config) (*Evaluator, error) {
	e := &Evaluator{
		name:     name,
		fallback: c.DefaultAction,
	}

	var err error

	e.allowedCountry, e.allowedCIDR, err = e.list(c.Allowlist)
	if err != nil {
		return nil, err
	}

	e.blockedCountry, e.blockedCIDR, err = e.list(c.Blocklist)
	return e, err
}

// AddLookup adds a lookup to the evaluator.
func (e *Evaluator) AddLookup(l lookup.Lookup) {
	e.lookups = append(e.lookups, l)
}

// Evaluate evaluates the state of the given IP.
func (e *Evaluator) Evaluate(addr string) (allowed bool, country string, err error) {
	ip := net.ParseIP(addr)
	if ip == nil {
		return false, "", fmt.Errorf("%s: invalid IP address: %s", e.name, addr)
	}

	//

	for _, block := range e.blockedCIDR {
		if block.Contains(ip) {
			return false, "", nil
		}
	}

	for _, lookup := range e.lookups {
		country, err = lookup.Country(ip)
		if err != nil {
			return false, "", fmt.Errorf("%s: country lookup: %w", e.name, err)
		}
	}

	if e.blockedCountry[country] {
		return false, country, nil
	}

	//

	for _, block := range e.allowedCIDR {
		if block.Contains(ip) {
			return true, country, nil
		}
	}

	if e.allowedCountry[country] {
		return true, country, nil
	}

	return e.fallback == DefaultActionAllow, country, nil
}

func (e *Evaluator) list(list []Rule) (map[string]bool, []*net.IPNet, error) {
	countries := make(map[string]bool)
	blocks := make([]*net.IPNet, 0)

	for _, r := range list {
		switch r.Type {
		case RuleTypeCountry:
			countries[strings.ToLower(r.Value)] = true
		case RuleTypeCIDR:
			_, block, err := net.ParseCIDR(r.Value)
			if err != nil {
				return nil, nil, fmt.Errorf("%s: invalid rule type: %s", e.name, r.Type)
			}

			blocks = append(blocks, block)
		default:
			return nil, nil, fmt.Errorf("%s: invalid rule type: %s", e.name, r.Type)
		}
	}

	return countries, blocks, nil
}
