package core

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"server/log"
	"shared/crypto"
	"strconv"
)

type Config struct {
	BindAddr       string
	BindPort       int
	AcceptAddr     string
	ServerIdentity string
	LogFile        string
	Fingerprint    string
	Silent         bool
	Reverse        bool
}

func (c *Config) Bind() string {
	return c.BindAddr + ":" + strconv.Itoa(c.BindPort)
}

func (s *Server) buildTLSConfig() (*tls.Config, error) {

	var bundle *crypto.Bundle
	var err error

	if s.config.ServerIdentity != "" {
		bundle, err = crypto.LoadBundle(s.config.ServerIdentity)
	} else {
		bundle, err = crypto.GenerateSelfSignedBundle(crypto.GonduitServerCN)
	}

	if err != nil {
		return nil, err
	}

	if bundle.IsSelfSigned() {
		log.Warnf("using self-signed certificate with fingerprint %s", bundle.Fingerprint())
	} else {
		log.Infof("using identity bundle at %s", s.config.ServerIdentity)
	}

	if s.config.Reverse {
		return crypto.WithVerification(bundle.NewTLSClientConfig(crypto.GonduitAppCN), s.acceptFingerprint), nil
	}

	return crypto.WithVerification(bundle.NewTLSServerConfig(), s.acceptFingerprint), nil

}

func (s *Server) acceptFingerprint(_ *x509.Certificate, fingerprint string, err error) error {

	// Return the original error if no fingerprint is configured
	if s.config.Fingerprint == "" {
		return err
	}

	// Accept matching fingerprint
	if s.config.Fingerprint == fingerprint {
		return nil
	}

	log.Errorf("rejecting untrusted certificate with fingerprint %s", fingerprint)

	return fmt.Errorf("untrusted certificate fingerprint: %s", fingerprint)

}
