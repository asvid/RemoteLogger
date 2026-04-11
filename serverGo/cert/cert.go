package cert

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

type Info struct {
	TLSCert     tls.Certificate
	Fingerprint string // "sha256/base64==" — paste into Config.CertPin on Android
	Dir         string // where cert.pem and key.pem are stored
}

// LoadOrGenerate loads the certificate from the user config dir, generating it
// on first run. The fingerprint is stable as long as the cert files exist.
func LoadOrGenerate() (*Info, error) {
	dir, err := configDir()
	if err != nil {
		return nil, err
	}

	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")

	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		if err := generate(certPath, keyPath); err != nil {
			return nil, err
		}
	}

	tlsCert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, err
	}

	fp, err := fingerprint(tlsCert)
	if err != nil {
		return nil, err
	}

	return &Info{
		TLSCert:     tlsCert,
		Fingerprint: fp,
		Dir:         dir,
	}, nil
}

func configDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "remotelogger")
	return dir, os.MkdirAll(dir, 0700)
}

// fingerprint computes SHA-256 of the SubjectPublicKeyInfo (SPKI) —
// the same value OkHttp's CertificatePinner and our custom trust manager use.
func fingerprint(c tls.Certificate) (string, error) {
	x509Cert, err := x509.ParseCertificate(c.Certificate[0])
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(x509Cert.RawSubjectPublicKeyInfo)
	return "sha256/" + base64.StdEncoding.EncodeToString(hash[:]), nil
}

func generate(certPath, keyPath string) error {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{Organization: []string{"RemoteLogger"}},
		NotBefore:    time.Now().Add(-time.Hour), // small backdating for clock skew
		NotAfter:     time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return err
	}
	if err := writePEM(certPath, "CERTIFICATE", certDER, 0644); err != nil {
		return err
	}

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return err
	}
	return writePEM(keyPath, "EC PRIVATE KEY", keyDER, 0600)
}

func writePEM(path, blockType string, der []byte, mode os.FileMode) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer f.Close()
	return pem.Encode(f, &pem.Block{Type: blockType, Bytes: der})
}
