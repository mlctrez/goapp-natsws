package gocert

import (
	"crypto/tls"
)

func DevTlsConfig(domain string) (config *tls.Config, err error) {
	var certResponse *CertificateResponse

	if certResponse, err = GenerateNats(domain, false); err != nil {
		return
	}

	config, err = certResponse.ServerTlsConfig(false)

	return
}
