package crypto

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"strings"
	"time"
)

// VerifyFunc is called when an untrusted certificate is encountered
type VerifyFunc func(cert *x509.Certificate, fingerprint string, err error) error

// WithVerification modifies a TLS config to enforce custom verification logic using a provided VerifyFunc
func WithVerification(config *tls.Config, verify VerifyFunc) *tls.Config {

	config = config.Clone()
	config.MinVersion = tls.VersionTLS13

	// Disable certificate verification to enable custom verification with VerifyPeerCertificate below
	config.InsecureSkipVerify = true //nolint:gosec
	config.SessionTicketsDisabled = true
	config.VerifyPeerCertificate = func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {

		if len(rawCerts) == 0 {
			return fmt.Errorf("no valid certificate presented by remote")
		}

		// Should never happen, but just in case
		if len(config.Certificates) == 0 {
			return fmt.Errorf("no certificates loaded")
		}

		cert, err := x509.ParseCertificate(rawCerts[0])

		if err != nil {
			return fmt.Errorf("failed to parse certificate: %w", err)
		}

		roots := config.RootCAs
		if roots == nil {
			roots = config.ClientCAs
		}

		// If roots are defined (not self-signed certificate), verify the remote cert
		if roots != nil {

			opts := x509.VerifyOptions{
				Roots:       roots,
				CurrentTime: time.Now(),
			}

			_, err = cert.Verify(opts)

			if err == nil {
				return nil
			}

			// Only allow call to verify for unknown authorities, hard reject everything else
			var unknownAuth x509.UnknownAuthorityError
			if !errors.As(err, &unknownAuth) {
				return err
			}

		}

		// Call the custom verify callback if provided, otherwise return the previous error (if any)
		if verify != nil {
			return verify(cert, Fingerprint(cert), err)
		}

		return err

	}

	return config

}

// Fingerprint returns the SHA-256 fingerprint of a certificate
func Fingerprint(cert *x509.Certificate) string {
	sum := sha256.Sum256(cert.Raw)
	var buf strings.Builder
	for _, b := range sum {
		_, _ = fmt.Fprintf(&buf, "%02X", b)
	}
	return buf.String()
}
