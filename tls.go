package gmi

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"path/filepath"
)
import "golang.org/x/crypto/ssh"
import kh "golang.org/x/crypto/ssh/knownhosts"

func dialTLS(ctx context.Context, u *url.URL) (*tls.Conn, error) {
	conn, err := tls.Dial("tcp", u.Host, nil)
	if err == nil {
		// standard verify success!
		return conn, nil
	}
	if u.Scheme != "gemini" {
		return nil, err
	}
	return dialGemini(ctx, u.Host, certFrom(err))
}
func dialGemini(ctx context.Context, capsule string, cert *x509.Certificate) (*tls.Conn, error) {
	if cert == nil {
		return nil, fmt.Errorf("TLS error is not in recovery set.")
	}
	isv := newMask(ctx, maskISVKey)
	knownCap := knownCapsules(ctx, capsule, cert, isv)

	cfg := &tls.Config{InsecureSkipVerify: true,
		MinVersion: tls.VersionTLS12,
		VerifyConnection: func(cs tls.ConnectionState) error {
			return recoveryVerify(cs, cert, isv, knownCap)
		},
	}
	return tls.Dial("tcp", capsule, cfg)
}

// verify which can toggle fallbacks
func recoveryVerify(cs tls.ConnectionState, cert *x509.Certificate, isv Mask, known bool) error {
	// WARNING: only for Gemini
	opts := x509.VerifyOptions{
		DNSName:       cs.ServerName,
		Intermediates: x509.NewCertPool(),
	}
	for _, pc := range cs.PeerCertificates[1:] {
		opts.Intermediates.AddCert(pc)
	}
	if isv.Has(AcceptUAE) || known && isv.Has(PromptUAE) {
		// treat self-signed cert as if root
		cert.IsCA = true
		opts.Roots = x509.NewCertPool()
		opts.Roots.AddCert(cert)
	}
	leaf := cs.PeerCertificates[0]
	if isv.Has(AcceptLCN) {
		// inject SAN
		leaf.DNSNames = append(leaf.DNSNames, cert.Subject.CommonName)
	}

	_, err := leaf.Verify(opts)
	return err
}
func knownCapsules(ctx context.Context, capsule string, cert *x509.Certificate, isv Mask) bool {
	if !isv.Has(PromptUAE) {
		return false
	}
	var err error
	if err = searchKnown(cert, capsule, "known_capsules"); err == nil {
		return true
	}
	if ke, ok := err.(*kh.KeyError); ok {
		if len(ke.Want) != 0 {
			return false
		}
		cpe := continueCapsulePrompt(ctx, capsule, cert, "known_capsules")
		if cpe == nil {
			return true
		}

	}
	return false
}
func continueCapsulePrompt(ctx context.Context, capsule string, cert *x509.Certificate, kcp string) error {
	//TODO prompt TOFU callback
	log.Printf("DEBUG TOFU prompt placeholder, PRETEND answer is Y for now")

	abs, err := filepath.Abs(kcp)
	if err != nil {
		return fmt.Errorf("Capsule prompt failed path, %w", err)
	}
	sshpk, err := ssh.NewPublicKey(cert.PublicKey)
	if err != nil {
		return fmt.Errorf("Capsule prompt failed new key, %w", err)
	}
	line := kh.Line([]string{capsule}, sshpk)
	file, err := os.OpenFile(abs, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 644)
	if err != nil {
		return fmt.Errorf("Capsule prompt failed file, %w", err)
	}
	defer file.Close()
	_, err = file.WriteString(line)
	if err != nil {
		return fmt.Errorf("Capsule prompt failed append, %w", err)
	}
	file.WriteString("\n")

	return nil
}
func searchKnown(cert *x509.Certificate, capsule string, kcp string) error {
	// known_hosts adapted as "known_capsules"
	sshpk, err := ssh.NewPublicKey(cert.PublicKey)
	if err != nil {
		log.Printf("DEBUG crt to ssh key failed, %v", err)
		return err
	}
	abs, err := filepath.Abs(kcp)
	if err != nil {
		log.Printf("DEBUG known_capsules path, %v", err)
		return err
	}
	hostKeyCallback, err := kh.New(abs)
	if err != nil {
		log.Printf("DEBUG callback not created, %v", err)
		return err
	}
	addr, err := net.ResolveTCPAddr("tcp", capsule)
	if err != nil {
		log.Printf("DEBUG resolve, %v", err)
		return err
	}
	err = hostKeyCallback(capsule, addr, sshpk)
	if err != nil {
		log.Printf("DEBUG known error, %v", err)
		return err
	}

	return nil
}
func certFrom(err error) *x509.Certificate {
	//TODO graceful when error is unsupported
	// supported errors are unknown-auth, commonname, expired
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
		return nil
	}
	return nil
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
	UAEReject
	PromptUAE
	AcceptUAE
)

func newMask(ctx context.Context, key string) Mask {
	// extract bit flags carried by context
	//todo sanity checks on ctx
	test := ctx.Value(key).(Mask)
	log.Printf("INFO ctx isv, %v", test)
	return test
}
func (m Mask) Set(flag Mask) Mask { return m | flag }
func (m Mask) Has(flag Mask) bool { return m&flag != 0 }
