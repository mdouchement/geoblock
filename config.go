package geoblock

import (
	"net/http"

	"github.com/mdouchement/geoblock/lookup"
)

// Rule data types.
const (
	RuleTypeCountry RuleType = "country"
	RuleTypeCIDR    RuleType = "cidr"
)

// Supported default actions.
const (
	DefaultActionAllow = "allow"
	DefaultActionBlock = "block"
)

type (
	// A Config defines the plugin configuration.
	Config struct {
		Enabled              bool            // Enable this plugin?
		AllowLetsEncrypt     bool            // Allow Let's Encrypt challenge path.
		Databases            []string        // Path to ip2location database files.
		DatabaseReaders      []lookup.Reader // Overrides Databases paths mostly for test purposes.
		DisallowedStatusCode int             // HTTP status code to return for disallowed requests.
		DefaultAction        string          // Default action to perform when there is no specified rule.
		Allowlist            []Rule
		Blocklist            []Rule
	}

	// A RuleType defines the type of a rule.
	RuleType string

	// A Rule is used to define if a request can be allowed or blocked.
	Rule struct {
		Type  RuleType
		Value string
	}
)

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		AllowLetsEncrypt:     true,
		DisallowedStatusCode: http.StatusForbidden,
		DefaultAction:        DefaultActionBlock,
		Blocklist: []Rule{
			{
				Type:  RuleTypeCIDR,
				Value: "127.0.0.0/8", // IPv4 loopback
			},
			{
				Type:  RuleTypeCIDR,
				Value: "10.0.0.0/8", // RFC1918
			},
			{
				Type:  RuleTypeCIDR,
				Value: "172.16.0.0/12", // RFC1918
			},
			{
				Type:  RuleTypeCIDR,
				Value: "192.168.0.0/16", // RFC1918
			},
			{
				Type:  RuleTypeCIDR,
				Value: "169.254.0.0/16", // RFC3927 link-local
			},
			{
				Type:  RuleTypeCIDR,
				Value: "::1/128", // IPv6 loopback
			},
			{
				Type:  RuleTypeCIDR,
				Value: "fe80::/10", // IPv6 link-local
			},
			{
				Type:  RuleTypeCIDR,
				Value: "fc00::/7", // IPv6 unique local addr
			},
		},
	}
}
