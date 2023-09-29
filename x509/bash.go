//go:build bashpki

// A reference implementation for those who want to customize their PKI.
// This is turned off in vanilla Fioctl builds, and can be enabled in a fork.

package x509

import (
	"os"
	"os/exec"

	"github.com/foundriesio/fioctl/subcommands"
)

type KeyStorage interface {
	configure()
}

func (s *fileStorage) configure() {}

func (s *hsmStorage) configure() {
	os.Setenv("HSM_MODULE", s.Module)
	os.Setenv("HSM_PIN", s.Pin)
	os.Setenv("HSM_TOKEN_LABEL", s.TokenLabel)
}

func run(script string, arg ...string) string {
	factoryCaKeyStorage.configure()
	arg = append([]string{"-s"}, arg...)
	cmd := exec.Command("/bin/sh", arg...)
	cmd.Stderr = os.Stderr
	in, err := cmd.StdinPipe()
	subcommands.DieNotNil(err, "Failed to start the shell")

	go func() {
		defer in.Close()
		_, err := in.Write([]byte(script))
		subcommands.DieNotNil(err, "Failed to pass the script to the shell")
	}()

	out, err := cmd.Output()
	subcommands.DieNotNil(err, "Failed to execute the shell script")
	return string(out)
}

func CreateFactoryCa(ou string) string {
	const script = `#!/bin/sh -e
## This script creates the offline private key, factory_ca.key, and x509 certficate, factory_ca.pem,
## owned by the customer that provides a chain of trust for all other certficates used by this factory.

if [ $# -ne 1 ] ; then
	echo "ERROR: $0 <ou>"
	exit 1
fi

ou=$1

cat >ca.cnf <<EOF
[req]
prompt = no
distinguished_name = dn
x509_extensions = ext

[dn]
CN = Factory-CA
OU = ${ou}

[ext]
basicConstraints=CA:TRUE
keyUsage = keyCertSign
extendedKeyUsage = critical, clientAuth, serverAuth
EOF

if [ -n "$HSM_MODULE" ] ; then
	lbl=${HSM_TOKEN_LABEL-device-gateway-root}
	pkcs11-tool --module $HSM_MODULE \
		--keypairgen --key-type EC:prime256v1 \
		--token-label $lbl \
		--id 01 \
		--label root-ca \
		--pin $HSM_PIN

	key="pkcs11:token=${lbl};object=root-ca;type=private;pin-value=$HSM_PIN"
	extra="-engine pkcs11 -keyform engine"
else
	openssl ecparam -genkey -name prime256v1 | openssl ec -out factory_ca.key
	key=factory_ca.key
	chmod 400 $key
fi
openssl req $extra -new -x509 -days 7300 -config ca.cnf -key "$key" -out factory_ca.pem -sha256
chmod 400 factory_ca.pem
rm ca.cnf
`
	run(script, ou)
	return readFile(FactoryCaCertFile)
}

func CreateDeviceCa(cn string, ou string) string {
	const script = `#!/bin/sh -e
## This is an optional script a customer can use to create a certificate
## capable of signing a certicate signing request from an LMP device.
## The certicate created here will be trusted by the Foundries device gateway.
## This is useful for creating CA owned by the customer for use in a manufacturing facility.

if [ $# -ne 3 ] ; then
	echo "ERROR: $0 <key-create> <cn> <ou>"
	exit 1
fi
key=$1
cn=$2
ou=$3

cat >ca.cnf <<EOF
[req]
prompt = no
distinguished_name = dn
x509_extensions = ext

[dn]
CN = ${cn}
OU = ${ou}

[ext]
keyUsage=critical, keyCertSign
basicConstraints=critical, CA:TRUE, pathlen:0
EOF

openssl ecparam -genkey -name prime256v1 | openssl ec -out $key
openssl req -new -config ca.cnf -key $key
chmod 400 $key
rm ca.cnf`
	csrPem := run(script, DeviceCaKeyFile, cn, ou)
	crtPem := signCaCsr("device-ca-*", csrPem)
	writeFile(DeviceCaCertFile, crtPem)
	return crtPem
}

func SignTlsCsr(csrPem string) string {
	const script = `#!/bin/sh -e
## This script signs the "tls-csr" returned when creating Factory certificates.
## This certificate are signed so that the devices trust the TLS connection with the Foundries device gateway.

if [ $# -ne 2 ] ; then
	echo "ERROR: $0 <tls csr> <tls crt>"
	exit 1
fi
csr=$1
crt=$2

dns=$(openssl req -text -noout -verify -in $csr | grep DNS:)
echo "signing with dns name: $dns" 1>&2

cat >server.ext <<EOF
keyUsage=critical, digitalSignature, keyEncipherment, keyAgreement
extendedKeyUsage=critical, serverAuth
subjectAltName=$dns
EOF

if [ -n "$HSM_MODULE" ] ; then
	lbl=${HSM_TOKEN_LABEL-device-gateway-root}
	key="pkcs11:token=${lbl};object=root-ca;type=private;pin-value=$HSM_PIN"
	extra="-CAkeyform engine -engine pkcs11"
else
	key=factory_ca.key
fi

openssl x509 -req -days 3650 $extra -in $csr -CAcreateserial \
	-extfile server.ext -CAkey "$key" -CA factory_ca.pem -sha256 -out $crt
rm server.ext factory_ca.srl || true`
	crtPem := signCsr(script, "tls-*", csrPem)
	writeFile(TlsCertFile, crtPem)
	return crtPem
}

func SignCaCsr(csrPem string) string {
	crtPem := signCaCsr("online-ca-*", csrPem)
	writeFile(OnlineCaCertFile, crtPem)
	return crtPem
}

func SignEl2GoCsr(csrPem string) string {
	return signCaCsr("el2g-*", csrPem)
}

func signCaCsr(tmpFileMask, csrPem string) string {
	const script = `#!/bin/sh -e
## This script signs "device ca" signing requests.
## The request may come from either the Factory (so that lmp-device-register will work)
## or locally with "create_device_ca" for manufacturing style device creation.

if [ $# -ne 2 ] ; then
	echo "ERROR: $0 <ca csr> <ca crt>"
	exit 1
fi
csr=$1
crt=$2

cat >ca.ext <<EOF
keyUsage=critical, keyCertSign
basicConstraints=critical, CA:TRUE, pathlen:0
EOF

if [ -n "$HSM_MODULE" ] ; then
	lbl=${HSM_TOKEN_LABEL-device-gateway-root}
	key="pkcs11:token=${lbl};object=root-ca;type=private;pin-value=$HSM_PIN"
	extra="-CAkeyform engine -engine pkcs11"
else
	key=factory_ca.key
fi

openssl x509 -req -days 3650 $extra -in $csr -CAcreateserial \
	-extfile ca.ext -CAkey "$key" -CA factory_ca.pem -sha256 -out $crt
rm ca.ext`
	return signCsr(script, tmpFileMask, csrPem)
}

func signCsr(script, tmpFileMask, csrPem string) string {
	csrFile, err := os.CreateTemp("", tmpFileMask+".csr")
	subcommands.DieNotNil(err)
	defer os.Remove(csrFile.Name())
	defer csrFile.Close()
	_, err = csrFile.Write([]byte(csrPem))
	subcommands.DieNotNil(err)

	crtFile, err := os.CreateTemp("", tmpFileMask+".crt")
	subcommands.DieNotNil(err)
	defer os.Remove(crtFile.Name())
	defer crtFile.Close()

	run(script, csrFile.Name(), crtFile.Name())
	return readFile(crtFile.Name())
}
