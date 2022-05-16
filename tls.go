package gmi

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"log"
	"net/url"
)

func dialTLS(ctx context.Context, u *url.URL) (*tls.Conn, error) {
	conn, err := tls.Dial("tcp", u.Host, nil)
	if err == nil {
		// standard verify success!
		return conn, nil
	}
	if u.Scheme != "gemini" {
		return nil, err
	}
	return dialGemini(ctx, u.Host, err)
}
func dialGemini(ctx context.Context, host string, err error) (*tls.Conn, error) {
	isv := newMask(ctx, maskISVKey)
	cert := certFrom(err)
	cfg := &tls.Config{InsecureSkipVerify: true,
		MinVersion: tls.VersionTLS12,
		VerifyConnection: func(cs tls.ConnectionState) error {
			return recoveryVerify(cs, cert, isv)
		},
	}
	return tls.Dial("tcp", host, cfg)
}

func certFrom(err error) *x509.Certificate {
	switch et := err.(type) {
	case x509.UnknownAuthorityError:
		uae, _ := err.(x509.UnknownAuthorityError)
		return uae.Cert

	case x509.HostnameError:
		hne, _ := err.(x509.HostnameError)
		log.Printf("DEBUG Name err cn: %v, h:%s, sz: %d",
			hne.Certificate.Subject.CommonName, hne.Host,
			len(hne.Certificate.DNSNames))
		if hne.Certificate.Subject.CommonName == hne.Host {
			return hne.Certificate
		}

	case x509.CertificateInvalidError:
		cie, _ := err.(x509.CertificateInvalidError)
		if cie.Reason == x509.Expired {
			log.Printf("DEBUG Expired cert, %s", cie.Detail)
			return cie.Cert
		}

	default:
		log.Printf("DEBUG Cert error type, %T", et)
		// unsupported cert error
		return nil
	}
	return nil
}

// verify that uses bitmask to toggle fallbacks
func recoveryVerify(cs tls.ConnectionState, ssc *x509.Certificate, isv Mask) error {
	// WARNING: only for Gemini
	opts := x509.VerifyOptions{
		DNSName:       cs.ServerName,
		Intermediates: x509.NewCertPool(),
	}
	for _, pc := range cs.PeerCertificates[1:] {
		opts.Intermediates.AddCert(pc)
	}
	if isv.Has(AcceptSSC) {
		// treat self-signed cert as if root
		ssc.IsCA = true
		opts.Roots = x509.NewCertPool()
		opts.Roots.AddCert(ssc)
	}
	leaf := cs.PeerCertificates[0]
	if isv.Has(AcceptLCN) {
		// inject SAN
		leaf.DNSNames = append(leaf.DNSNames, ssc.Subject.CommonName)
	}

	_, err := leaf.Verify(opts)
	return err
}

const maskISVKey = "InsecureSkipVerify"

type Mask uint16

const (
	None Mask = 1 << iota
	LCNReject
	LCNPrompt
	AcceptLCN
	CIEReject
	CIEPrompt
	AcceptCIE
	SSCReject
	SSCPrompt
	AcceptSSC
)

func newMask(ctx context.Context, key string) Mask {
	// isv bit flags passed via context
	var m Mask
	//todo sanity checks on ctx
	test := m.From(ctx, key)
	log.Printf("INFO ctx isv, %v", test)
	return test
}
func (m Mask) From(ctx context.Context, key string) Mask { return ctx.Value(key).(Mask) }
func (m Mask) Set(flag Mask) Mask                        { return m | flag }
func (m Mask) Has(flag Mask) bool                        { return m&flag != 0 }

// TODO
//      implement TOFU as described by the spec
//      maybe ref https://pkg.go.dev/golang.org/x/crypto/ssh/knownhosts
//
