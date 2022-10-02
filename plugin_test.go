package geoblock_test

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/mdouchement/geoblock"
	"github.com/stretchr/testify/assert"
)

type noopHandler struct{}

func (n noopHandler) ServeHTTP(rw http.ResponseWriter, _ *http.Request) {
	rw.WriteHeader(http.StatusTeapot)
}

func TestCreateConfig(t *testing.T) {
	c := &geoblock.Config{
		AllowLetsEncrypt:     true,
		DisallowedStatusCode: http.StatusForbidden,
		DefaultAction:        geoblock.DefaultActionBlock,
		Blocklist: []geoblock.Rule{
			{
				Type:  geoblock.RuleTypeCIDR,
				Value: "127.0.0.0/8", // IPv4 loopback
			},
			{
				Type:  geoblock.RuleTypeCIDR,
				Value: "10.0.0.0/8", // RFC1918
			},
			{
				Type:  geoblock.RuleTypeCIDR,
				Value: "172.16.0.0/12", // RFC1918
			},
			{
				Type:  geoblock.RuleTypeCIDR,
				Value: "192.168.0.0/16", // RFC1918
			},
			{
				Type:  geoblock.RuleTypeCIDR,
				Value: "169.254.0.0/16", // RFC3927 link-local
			},
			{
				Type:  geoblock.RuleTypeCIDR,
				Value: "::1/128", // IPv6 loopback
			},
			{
				Type:  geoblock.RuleTypeCIDR,
				Value: "fe80::/10", // IPv6 link-local
			},
			{
				Type:  geoblock.RuleTypeCIDR,
				Value: "fc00::/7", // IPv6 unique local addr
			},
		},
	}

	assert.Equal(t, c, geoblock.CreateConfig())
}

func TestPlugin_ServeHTTP(t *testing.T) {
	c := geoblock.CreateConfig()
	c.Enabled = true
	c.Databases = []string{
		"IP2LOCATION-LITE-DB1.BIN",
		"IP2LOCATION-LITE-DB1.IPV6.BIN",
	}
	c.Allowlist = append(c.Allowlist, geoblock.Rule{Type: geoblock.RuleTypeCountry, Value: "fr"})

	err := check(c)
	if err != nil {
		assert.FailNow(t, err.Error())
	}

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

func check(c *geoblock.Config) error {
	for _, dbname := range c.Databases {
		_, err := os.Stat(dbname)
		if err == nil || os.IsExist(err) {
			continue
		}

		err = download(
			fmt.Sprintf("https://download.ip2location.com/lite/%s.ZIP", dbname),
			dbname,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func download(url, filename string) error {
	res, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("downloading db failed: %w", err)
	}
	defer res.Body.Close()

	payload, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("downloading db failed: %w", err)
	}

	var r io.Reader = bytes.NewReader(payload)
	if strings.HasSuffix(strings.ToLower(url), ".zip") {
		r = nil

		codec, err := zip.NewReader(bytes.NewReader(payload), res.ContentLength)
		if err != nil {
			return fmt.Errorf("creating zip reader: %w", err)
		}

		for _, file := range codec.File {
			if !strings.HasSuffix(strings.ToLower(file.Name), ".bin") {
				continue
			}

			f, err := file.Open()
			if err != nil {
				return fmt.Errorf("opening zip db file: %w", err)
			}
			defer f.Close()

			r = f
		}
	}

	if r == nil {
		return errors.New("db file not found in downloaded archive")
	}

	//

	w, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer w.Close()

	_, err = io.Copy(w, r)
	if err != nil {
		return fmt.Errorf("copy db to file: %w", err)
	}

	return w.Sync()
}
