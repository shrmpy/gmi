package gmi

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	//"os"
	//"time"
)

// TODO don't allow the skip-verify option of TLS config
//      implement TOFU as described by the spec
//      maybe ref https://pkg.go.dev/golang.org/x/crypto/ssh/knownhosts
/*
   ////conn, err := tls.Dial("tcp", u.Host, &tls.Config{InsecureSkipVerify: true})
   ca, err := newClientConfig(CA_ROOTS)
   if err != nil {
           return "", err
   }
*/

func loadRootCA(file string) (*x509.CertPool, error) {
	pemBytes, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM(pemBytes)
	if !ok {
		return roots, fmt.Errorf("failed to parse root certificate")
	}
	//caset := tls.NewCASet()
	//if caset.SetFromPEM(pemBytes) {
	//	return caset, nil
	//}
	return nil, fmt.Errorf("Unable to decode root CA set")
}

func newClientConfig(rootCAPath string) (*tls.Config, error) {
	rootca, err := loadRootCA(rootCAPath)
	if err != nil {
		return nil, err
	}

	/*
		urandom, err := os.Open("/dev/urandom")
		if err != nil {
			return nil, err
		}*/

	return &tls.Config{
		//Rand:    urandom,
		//Time:    time.Seconds,
		RootCAs: rootca,
	}, nil
}
