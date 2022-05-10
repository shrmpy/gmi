package gmi

import (
	"fmt"
	"net/url"
	"strings"
)

// format the URL for Gemini scheme
func Format(raw string, referer string) (*url.URL, error) {
	//TODO make unit tests to prove we follow
	//     https://gemini.circumlunar.space/docs/specification.gmi
	var (
		err error
		lu  *url.URL
		tmp = raw
		rfr = &url.URL{Scheme: "gemini", Host: ":1965"}
	)
	if strings.HasPrefix(referer, "gemini://") {
		if rfr, err = url.Parse(referer); err != nil {
			return &url.URL{}, err
		}
	}
	// b) no scheme, no host, relative path (causes empty scheme/host result)
	//    (are dot paths allowed in links?)
	if strings.HasPrefix(raw, "/") {
		tmp = fmt.Sprintf("gemini://%s%s", rfr.Host, raw)
	} else if foundAt := strings.Index(raw, ":/"); foundAt == -1 {
		// a) no scheme, host without port (causes empty Parse result)
		tmp = fmt.Sprintf("gemini://%s", raw)
	}

	if lu, err = url.Parse(tmp); err != nil {
		return &url.URL{}, fmt.Errorf("Error parsing URL! %v", err)
	}
	if !lu.IsAbs() {
		// relative?
		lu.Scheme = rfr.Scheme
		if lu.Hostname() == "" {
			// is-relative
			lu.Host = rfr.Host
		}
	}
	if lu.Port() == "" && lu.Scheme == "gemini" {
		// be unambiguous for port
		lu.Host += ":1965"
	}

	return lu, nil
}
