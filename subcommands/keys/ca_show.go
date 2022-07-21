package keys

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/subcommands"
)

var prettyFormat bool

func init() {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show what certificates are known to the factory",
		Run:   doShowCA,
	}
	caCmd.AddCommand(cmd)
	cmd.Flags().BoolVarP(&prettyFormat, "pretty", "", false, "Display human readable output of each certificate")
}

func doShowCA(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	logrus.Debugf("Showing certs for %s", factory)

	resp, err := api.FactoryGetCA(factory)
	subcommands.DieNotNil(err)

	fmt.Println("## Factory root certificate")
	if prettyFormat {
		prettyPrint(resp.RootCrt)
	} else {
		fmt.Println(resp.RootCrt)
	}
	fmt.Println("## Server TLS Certificate")
	if prettyFormat {
		prettyPrint(resp.TlsCrt)
	} else {
		fmt.Println(resp.TlsCrt)
	}
	fmt.Println("\n## Device Authentication Certificate(s)")
	if prettyFormat {
		prettyPrint(resp.CaCrt)
	} else {
		fmt.Println(resp.CaCrt)
	}
}

func keyUsage(val asn1.BitString) string {
	vals := ""
	if val.At(0) != 0 {
		vals += "digitalSignature "
	}
	if val.At(1) != 0 {
		vals += "nonRepudiation "
	}
	if val.At(2) != 0 {
		vals += "keyEncipherment "
	}
	if val.At(3) != 0 {
		vals += "dataEncipherment "
	}
	if val.At(4) != 0 {
		vals += "keyAgreement "
	}
	if val.At(5) != 0 {
		vals += "keyCertSign "
	}
	if val.At(6) != 0 {
		vals += "cRLSign "
	}
	if val.At(7) != 0 {
		vals += "encipherOnly "
	}
	if val.At(8) != 0 {
		vals += "decipherOnly "
	}
	return vals
}

func extKeyUsage(ext []x509.ExtKeyUsage) string {
	vals := ""
	for _, u := range ext {
		switch u {
		case x509.ExtKeyUsageAny:
			vals += "KeyUsageAny "
		case x509.ExtKeyUsageServerAuth:
			vals += "ServerAuth "
		case x509.ExtKeyUsageClientAuth:
			vals += "ClientAuth "
		default:
			vals += fmt.Sprintf("Unknown(%d) ", u)
		}
	}
	return vals
}

func prettyPrint(cert string) {
	for len(cert) > 0 {
		block, remaining := pem.Decode([]byte(cert))
		if block == nil {
			// could be excessive whitespace
			cert = strings.TrimSpace(string(remaining))
			continue
		}
		cert = string(remaining)
		c, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			fmt.Println("Failed to parse certificate:" + err.Error())
			continue
		}
		fmt.Println("Certificate:")
		fmt.Println("\tVersion:", c.Version)
		fmt.Println("\tSerial Number:", c.SerialNumber)
		fmt.Println("\tSignature Algorithm:", c.SignatureAlgorithm)
		fmt.Println("\tIssuer:", c.Issuer)
		fmt.Println("\tValidity")
		fmt.Println("\t\tNot Before:", c.NotBefore)
		fmt.Println("\t\tNot After:", c.NotAfter)
		fmt.Println("\tSubject:", c.Subject)
		fmt.Println("\tSubject Public Key Info")
		switch pub := c.PublicKey.(type) {
		case *ecdsa.PublicKey:
			fmt.Println("\t\tNIST CURVE:", pub.Curve.Params().Name)
			fmt.Print("\t\t\t")
			for idx, b := range elliptic.Marshal(pub.Curve, pub.X, pub.Y) {
				fmt.Printf("%02x:", b)
				if (idx+1)%15 == 0 {
					fmt.Println()
					fmt.Print("\t\t\t")
				}
			}
			fmt.Println()
		default:
			fmt.Println("Failed to read public key")
		}
		fmt.Println("\tIs CA:", c.IsCA)
		fmt.Println("\tExtensions:")
		for _, ext := range c.Extensions {
			if ext.Id.String() == "2.5.29.15" {
				fmt.Print("\t\tx509v3 Key Usage: ")
				if ext.Critical {
					fmt.Print("(critical)")
				}
				fmt.Println()
				var val asn1.BitString
				if _, err := asn1.Unmarshal(ext.Value, &val); err != nil {
					fmt.Println(err)
				} else {
					fmt.Println("\t\t\t", keyUsage(val))
				}
			} else if ext.Id.String() == "2.5.29.19" {
				fmt.Print("\t\tx509v3 Basic Constraints: ")
				if ext.Critical {
					fmt.Print("(critical)")
				}
				fmt.Println()
				if c.IsCA {
					fmt.Println("\t\t\tCA:TRUE MaxPath ", c.MaxPathLen)
				} else {
					fmt.Println("\t\t\tCA:FALSE")
				}
			} else if ext.Id.String() == "2.5.29.37" {
				fmt.Print("\t\tx509v3 Extended Key Usage: ")
				if ext.Critical {
					fmt.Print("(critical)")
				}
				fmt.Println()
				fmt.Println("\t\t\t", extKeyUsage(c.ExtKeyUsage))
			} else if ext.Id.String() == "2.5.29.17" {
				fmt.Print("\t\tx509v3 Subject Name Alternative: ")
				if ext.Critical {
					fmt.Print("(critical)")
				}
				fmt.Println()
				for _, name := range c.DNSNames {
					fmt.Println("\t\t\tDNS:", name)
				}
				for _, name := range c.IPAddresses {
					fmt.Println("\t\t\tIP:", name)
				}
				for _, name := range c.URIs {
					fmt.Println("\t\t\tURI:", name)
				}
				for _, name := range c.EmailAddresses {
					fmt.Println("\t\t\tEmail:", name)
				}
			} else {
				fmt.Println("Unknown OID", ext.Id.String())
			}
		}
	}
}
