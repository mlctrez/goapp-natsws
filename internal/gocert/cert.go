package gocert

import (
	"crypto/tls"
	"crypto/x509"
)

type CertificateResponse struct {
	CertificatePem          string `json:"crt_file" xml:",cdata"`
	CertificateKey          string `json:"key_file" xml:",cdata"`
	CertificateAuthorityPem string `json:"registry_ca" xml:",cdata"`
}

func (cr *CertificateResponse) Certificate() (tls.Certificate, error) {
	return tls.X509KeyPair([]byte(cr.CertificatePem), []byte(cr.CertificateKey))
}

func (cr *CertificateResponse) ClientTlsConfig() (config *tls.Config, err error) {

	certificate, err := cr.Certificate()
	if err != nil {
		return nil, err
	}

	config = &tls.Config{}

	config.Certificates = []tls.Certificate{certificate}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM([]byte(cr.CertificateAuthorityPem))

	config.RootCAs = caCertPool

	return config, nil

}

func (cr *CertificateResponse) ServerTlsConfig(clientCert bool) (config *tls.Config, err error) {

	certificate, err := cr.Certificate()
	if err != nil {
		return nil, err
	}

	config = &tls.Config{}

	config.Certificates = []tls.Certificate{certificate}

	if clientCert {
		clientCAs := x509.NewCertPool()
		clientCAs.AppendCertsFromPEM([]byte(cr.CertificateAuthorityPem))

		config.ClientAuth = tls.RequireAndVerifyClientCert
		config.ClientCAs = clientCAs
	}

	return config, nil

}
