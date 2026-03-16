package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"shared/util"
	"time"
)

const (
	defaultCertValidity = 10 * 365 * 24 * time.Hour
	pemTypeCert         = "CERTIFICATE"
	pemTypeKey          = "PRIVATE KEY"
	pemTypeECKey        = "EC PRIVATE KEY"
	gonduitCACN         = "gonduit-ca"
	GonduitServerCN     = "gonduit-server"
	GonduitAppCN        = "gonduit-app"
)

// Bundle holds a parsed CA cert alongside a signed tls.Certificate with chain
type Bundle struct {
	tlsCert    tls.Certificate
	caCert     *x509.Certificate
	selfSigned bool
}

func (bundle *Bundle) IsSelfSigned() bool {
	return bundle.selfSigned
}

func (bundle *Bundle) Fingerprint() string {
	return Fingerprint(bundle.tlsCert.Leaf)
}

// NewTLSServerConfig returns a tls.Config suitable for a gRPC server
func (bundle *Bundle) NewTLSServerConfig() *tls.Config {

	return &tls.Config{
		Certificates: []tls.Certificate{bundle.tlsCert},
		ClientCAs:    bundle.certPool(),
		ClientAuth:   tls.RequestClientCert,
		MinVersion:   tls.VersionTLS13,
	}

}

// NewTLSClientConfig returns a tls.Config suitable for a gRPC client (or reverse server dial)
func (bundle *Bundle) NewTLSClientConfig(serverName string) *tls.Config {

	return &tls.Config{
		ServerName:   serverName,
		Certificates: []tls.Certificate{bundle.tlsCert},
		RootCAs:      bundle.certPool(),
		MinVersion:   tls.VersionTLS13,
	}

}

func (bundle *Bundle) certPool() *x509.CertPool {
	if bundle.selfSigned {
		return nil
	}
	pool := x509.NewCertPool()
	pool.AddCert(bundle.caCert)
	return pool
}

// GenerateCA creates a new ECDSA CA key pair and self-signed certificate
func GenerateCA(subject pkix.Name) (*x509.Certificate, *ecdsa.PrivateKey, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	serial, err := randomSerial()
	if err != nil {
		return nil, nil, err
	}

	template := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               subject,
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(defaultCertValidity),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, nil, err
	}

	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, nil, err
	}

	return cert, key, nil
}

// SignCert creates a new ECDSA key pair and signs a certificate with the given CA
func SignCert(subject pkix.Name, ca *x509.Certificate, caKey *ecdsa.PrivateKey) (*x509.Certificate, *ecdsa.PrivateKey, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	serial, err := randomSerial()
	if err != nil {
		return nil, nil, err
	}

	template := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               subject,
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(defaultCertValidity),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		DNSNames:              []string{subject.CommonName},
		BasicConstraintsValid: true,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, ca, &key.PublicKey, caKey)
	if err != nil {
		return nil, nil, err
	}

	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, nil, err
	}

	return cert, key, nil
}

func GenerateBundles(serverPath, appPath string) error {

	ca, caKey, err := GenerateCA(pkix.Name{CommonName: gonduitCACN})
	if err != nil {
		return fmt.Errorf("generate CA: %w", err)
	}

	serverCert, serverKey, err := SignCert(pkix.Name{CommonName: GonduitServerCN}, ca, caKey)
	if err != nil {
		return fmt.Errorf("sign server cert: %w", err)
	}

	appCert, appKey, err := SignCert(pkix.Name{CommonName: GonduitAppCN}, ca, caKey)
	if err != nil {
		return fmt.Errorf("sign app cert: %w", err)
	}

	if err = SaveBundle(serverPath, serverCert, ca, serverKey); err != nil {
		return fmt.Errorf("save server bundle: %w", err)
	}

	if err = SaveBundle(appPath, appCert, ca, appKey); err != nil {
		return fmt.Errorf("save app bundle: %w", err)
	}

	return nil
}

// SaveBundle writes a PEM bundle to path containing (in order) the signed certificate, the CA certificate (chain) and the private key
func SaveBundle(path string, cert *x509.Certificate, ca *x509.Certificate, key *ecdsa.PrivateKey) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer util.CloseFile(f)

	// Leaf cert
	if err = pem.Encode(f, &pem.Block{Type: pemTypeCert, Bytes: cert.Raw}); err != nil {
		return err
	}

	// CA cert (chain)
	if err = pem.Encode(f, &pem.Block{Type: pemTypeCert, Bytes: ca.Raw}); err != nil {
		return err
	}

	// Private key
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return err
	}
	return pem.Encode(f, &pem.Block{Type: pemTypeKey, Bytes: keyDER})
}

// LoadBundle reads a PEM bundle saved with SaveBundle from path and returns a Bundle ready for use in tls.Config
func LoadBundle(path string) (*Bundle, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var certDERs [][]byte
	var keyDER []byte

	for len(data) > 0 {
		var block *pem.Block
		block, data = pem.Decode(data)
		if block == nil {
			break
		}
		switch block.Type {
		case pemTypeCert:
			certDERs = append(certDERs, block.Bytes)
		case pemTypeKey:
			keyDER = block.Bytes
		case pemTypeECKey:
			keyDER = block.Bytes
		}
	}

	if len(certDERs) < 2 {
		return nil, errors.New("bundle must contain at least two certificates (leaf + CA)")
	}

	if keyDER == nil {
		return nil, errors.New("bundle missing private key")
	}

	leafCert, err := x509.ParseCertificate(certDERs[0])
	if err != nil {
		return nil, err
	}

	caCert, err := x509.ParseCertificate(certDERs[1])
	if err != nil {
		return nil, err
	}

	key, err := x509.ParsePKCS8PrivateKey(keyDER)
	if err != nil {
		return nil, err
	}

	tlsCert := tls.Certificate{
		Certificate: certDERs,
		PrivateKey:  key,
		Leaf:        leafCert,
	}

	return &Bundle{tlsCert: tlsCert, caCert: caCert}, nil
}

func GenerateSelfSignedBundle(cn string) (*Bundle, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	serial, err := randomSerial()
	if err != nil {
		return nil, err
	}

	template := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: cn},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(defaultCertValidity),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		DNSNames:              []string{cn},
		BasicConstraintsValid: true,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}

	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, err
	}

	return &Bundle{
		tlsCert: tls.Certificate{
			Certificate: [][]byte{der},
			PrivateKey:  key,
			Leaf:        leaf,
		},
		caCert:     leaf,
		selfSigned: true,
	}, nil
}

func randomSerial() (*big.Int, error) {
	return rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
}
