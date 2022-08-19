package tlsutil

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"path/filepath"
)

func LoadCert(certDirPath string) (*tls.Certificate, error) {
	cert, err := tls.LoadX509KeyPair(filepath.Join(certDirPath, "tls.crt"), filepath.Join(certDirPath, "tls.key"))
	if err != nil {
		return nil, err
	}
	return &cert, nil
}

func LoadCACert(certDirPath string) (*x509.CertPool, error) {
	caCertData, err := ioutil.ReadFile(filepath.Join(certDirPath, "ca.crt"))
	if err != nil {
		return nil, fmt.Errorf("read CA cert file: %s", err)
	}
	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(caCertData)
	return certPool, nil
}
