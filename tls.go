package gmi

import (
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"fmt"
	"net/url"
)


func dialTLS(u *url.URL) (*tls.Conn, error) {
	if u.Scheme != "gemini" {
		return tls.Dial("tcp", u.Host, nil)
	}
	return dialGemini(u)
}
func dialGemini(u *url.URL) (*tls.Conn, error) {
	conn, err := tls.Dial("tcp", u.Host, nil)
	if err == nil {
		// normal verify is fine
		return conn, nil
	}
	e, ok := err.(x509.UnknownAuthorityError)
	if !ok {
		return nil, fmt.Errorf("Not-implemented: %v", err)
	}
	// TODO maintain known-hosts
	// dial for self-signed certs
	cfg := selfSignedConfig(e.Cert)
	return tls.Dial("tcp", u.Host, cfg)
}
func selfSignedConfig(selfcrt *x509.Certificate) *tls.Config {
	// WARNING: only for Gemini
	return &tls.Config{InsecureSkipVerify: true,
		MinVersion: tls.VersionTLS12,
		VerifyConnection: func(cs tls.ConnectionState) error {
			opts := x509.VerifyOptions{
				DNSName:       cs.ServerName,
				Intermediates: x509.NewCertPool(),
			}
			for _, cert := range cs.PeerCertificates[1:] {
				opts.Intermediates.AddCert(cert)
			}
			// make self-signed cert behave as a rootCA
			root := selfcrt
			root.IsCA = true
			opts.Roots = x509.NewCertPool()
			opts.Roots.AddCert(root)

			_, err := cs.PeerCertificates[0].Verify(opts)
			return err
		},
	}
}

// TODO
//      implement TOFU as described by the spec
//      maybe ref https://pkg.go.dev/golang.org/x/crypto/ssh/knownhosts
//
