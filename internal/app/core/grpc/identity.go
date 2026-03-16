package grpc

import (
	"fmt"
	"shared/crypto"
	"time"
)

type Identity struct {
	SelfSigned   bool
	CommonName   string
	SANs         []string
	Issuer       string
	NotBefore    time.Time
	NotAfter     time.Time
	Fingerprint  string
	SerialNumber string
}

func (m *Manager) Identity() (Identity, error) {

	if m.tlsConfig == nil {
		return Identity{}, fmt.Errorf("no identity loaded")
	}

	if len(m.tlsConfig.Certificates) == 0 {
		return Identity{}, fmt.Errorf("no certificates loaded")
	}

	cert := m.tlsConfig.Certificates[0].Leaf

	// Collect SANs
	sans := make([]string, 0, len(cert.DNSNames)+len(cert.IPAddresses))
	sans = append(sans, cert.DNSNames...)
	for _, ip := range cert.IPAddresses {
		sans = append(sans, ip.String())
	}

	return Identity{
		SelfSigned:   m.selfSigned,
		CommonName:   cert.Subject.CommonName,
		SANs:         sans,
		Issuer:       cert.Issuer.CommonName,
		NotBefore:    cert.NotBefore,
		NotAfter:     cert.NotAfter,
		Fingerprint:  crypto.Fingerprint(cert),
		SerialNumber: cert.SerialNumber.String(),
	}, nil

}

func (m *Manager) IsSelfSignedIdentity() bool {
	return m.selfSigned
}

func (m *Manager) HasIdentity() bool {
	return m.tlsConfig != nil
}

func (m *Manager) UseSelfSignedIdentity() error {

	if m.tlsConfig != nil {
		return fmt.Errorf("identity already loaded")
	}

	bundle, err := crypto.GenerateSelfSignedBundle(crypto.GonduitAppCN)

	if err != nil {
		return err
	}

	m.tlsConfig = bundle.NewTLSClientConfig(crypto.GonduitServerCN)
	m.selfSigned = true

	return nil

}

func (m *Manager) LoadIdentity(path string) error {

	if m.tlsConfig != nil {
		return fmt.Errorf("identity already loaded")
	}
	
	bundle, err := crypto.LoadBundle(path)

	if err != nil {
		return err
	}

	m.tlsConfig = bundle.NewTLSClientConfig(crypto.GonduitServerCN)
	m.selfSigned = false

	return nil

}

func (m *Manager) UnloadIdentity() error {

	if m.tlsConfig == nil {
		return fmt.Errorf("no identity loaded")
	}

	m.tlsConfig = nil

	return nil

}
