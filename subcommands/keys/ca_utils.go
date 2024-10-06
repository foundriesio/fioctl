package keys

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
)

func parseCertList(pemData string) (certs []*x509.Certificate) {
	for len(pemData) > 0 {
		block, remaining := pem.Decode([]byte(pemData))
		if block == nil {
			// could be excessive whitespace
			if pemData = strings.TrimSpace(string(remaining)); len(pemData) == len(remaining) {
				fmt.Println("Failed to parse remaining certificates: invalid PEM data")
				break
			}
			continue
		}
		pemData = string(remaining)
		c, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			fmt.Println("Failed to parse certificate:" + err.Error())
			continue
		}
		certs = append(certs, c)
	}
	return
}
