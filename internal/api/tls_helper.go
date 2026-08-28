package api

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"time"
)

// LoadOrCreateTLSConfig intelligently loads system certificates (supporting PEM and DER formats), or automatically generates a self-signed certificate on the fly
func LoadOrCreateTLSConfig() (*tls.Config, error) {
	// 1. Attempt to load and parse from common system paths
	certFiles := []string{"/etc/uhttpd.crt", "/etc/ssl/certs/uhttpd.crt", "/etc/nginx/uhttpd.crt"}
	keyFiles := []string{"/etc/uhttpd.key", "/etc/ssl/certs/uhttpd.key", "/etc/nginx/uhttpd.key"}

	for i := range certFiles {
		certData, errCert := os.ReadFile(certFiles[i])
		keyData, errKey := os.ReadFile(keyFiles[i])
		if errCert == nil && errKey == nil && len(certData) > 0 && len(keyData) > 0 {
			cert, err := parseCertAndKey(certData, keyData)
			if err == nil {
				log.Printf("[API] Successfully loaded system TLS certificate from %s", certFiles[i])
				return &tls.Config{
					Certificates: []tls.Certificate{*cert},
				}, nil
			}
		}
	}

	// 2. If system certs do not exist or have proprietary formats, generate standard self-signed ECDSA certificate
	log.Println("[API] Generating self-signed TLS certificate for HTTPS...")
	cert, err := generateSelfSignedCert()
	if err != nil {
		return nil, fmt.Errorf("generate self-signed cert failed: %w", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{*cert},
	}, nil
}

// parseCertAndKey parses PEM and DER format certificates and private keys
func parseCertAndKey(certBytes, keyBytes []byte) (*tls.Certificate, error) {
	// 1. Attempt parsing directly as PEM
	cert, err := tls.X509KeyPair(certBytes, keyBytes)
	if err == nil {
		return &cert, nil
	}

	// 2. Attempt converting DER format to PEM
	var pemCert, pemKey []byte

	// Check if cert is already PEM
	if block, _ := pem.Decode(certBytes); block == nil {
		pemCert = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	} else {
		pemCert = certBytes
	}

	// Check if key is already PEM
	if block, _ := pem.Decode(keyBytes); block == nil {
		// Attempt parsing various DER private key formats
		if _, err := x509.ParsePKCS8PrivateKey(keyBytes); err == nil {
			pemKey = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes})
		} else if _, err := x509.ParseECPrivateKey(keyBytes); err == nil {
			pemKey = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})
		} else if _, err := x509.ParsePKCS1PrivateKey(keyBytes); err == nil {
			pemKey = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyBytes})
		} else {
			// Generic PRIVATE KEY wrapper
			pemKey = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes})
		}
	} else {
		pemKey = keyBytes
	}

	convertedCert, err := tls.X509KeyPair(pemCert, pemKey)
	if err == nil {
		return &convertedCert, nil
	}

	return nil, err
}

// generateSelfSignedCert generates a self-signed ECDSA P-256 certificate on the fly
func generateSelfSignedCert() (*tls.Certificate, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	notBefore := time.Now().Add(-10 * time.Minute)
	notAfter := notBefore.Add(3650 * 24 * time.Hour) // 10-year validity

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"ParentControl Guard"},
			CommonName:   "OpenWrt Router",
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("192.168.0.1"), net.ParseIP("192.168.1.1"), net.ParseIP("192.168.0.110")},
		DNSNames:              []string{"localhost", "openwrt.lan", "router.lan", "istoreos.lan"},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	privBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return nil, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes})

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}

	return &cert, nil
}
