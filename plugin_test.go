package geoblock_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mdouchement/geoblock"
	"github.com/stretchr/testify/assert"
)

type noopHandler struct{}

func (n noopHandler) ServeHTTP(rw http.ResponseWriter, _ *http.Request) {
	rw.WriteHeader(http.StatusTeapot)
}

func TestTestPlugin_ServeHTTP(t *testing.T) {
	c := geoblock.CreateConfig()
	c.Enabled = true
	c.Databases = []string{
		"IP2LOCATION-LITE-DB1.BIN",
		"IP2LOCATION-LITE-DB1.IPV6.BIN",
	}
	c.Allowlist = append(c.Allowlist, geoblock.Rule{Type: geoblock.RuleTypeCountry, Value: "fr"})

	plugin, err := geoblock.New(nil, new(noopHandler), c, "geoblock")
	assert.NoError(t, err)

	//

	tests := []struct {
		header string
		ip     string
		status int
	}{
		{
			header: "X-Forwarded-For",
			ip:     "127.0.0.1",
			status: http.StatusForbidden,
		},
		{
			header: "X-Real-IP",
			ip:     "127.0.0.1",
			status: http.StatusForbidden,
		},
		{
			header: "X-Forwarded-For",
			ip:     "1.1.1.1",            // US
			status: http.StatusForbidden, // default_action
		},
		{
			header: "X-Forwarded-For",
			ip:     "2606:4700:4700::1111", // US
			status: http.StatusForbidden,   // default_action
		},
		{
			header: "X-Forwarded-For",
			ip:     "80.67.169.12", // FR
			status: http.StatusTeapot,
		},
		{
			header: "X-Forwarded-For",
			ip:     "2001:910:800::12", // FR
			status: http.StatusTeapot,
		},
	}

	for _, test := range tests {
		req := httptest.NewRequest(http.MethodGet, "/"+test.header, nil)
		req.Header.Set(test.header, test.ip)

		rr := httptest.NewRecorder()
		plugin.ServeHTTP(rr, req)

		assert.Equal(t, test.status, rr.Code)
	}
}
