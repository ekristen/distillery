package cosign

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
)

func ParsePublicKey(pemEncodedPubKey []byte) (*ecdsa.PublicKey, error) {
	block, _ := pem.Decode(pemEncodedPubKey)
	if block == nil || (block.Type != "PUBLIC KEY" && block.Type != "CERTIFICATE") {
		return nil, errors.New("failed to decode PEM block containing public key or certificate")
	}

	var ecdsaPub *ecdsa.PublicKey

	if block.Type == "PUBLIC KEY" { //nolint:staticcheck
		pub, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, err
		}

		var ok bool
		ecdsaPub, ok = pub.(*ecdsa.PublicKey)
		if !ok {
			return nil, errors.New("not ECDSA public key")
		}
	} else if block.Type == "CERTIFICATE" {
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}

		var ok bool
		ecdsaPub, ok = cert.PublicKey.(*ecdsa.PublicKey)
		if !ok {
			return nil, errors.New("not ECDSA public key")
		}
	}

	return ecdsaPub, nil
}

func HashData(data []byte) []byte {
	hasher := sha256.New()
	hasher.Write(data)
	return hasher.Sum(nil)
}

// VerifySignature verifies the signature of the data using the provided ECDSA public key.
func VerifySignature(pubKey *ecdsa.PublicKey, hash, signature []byte) (bool, error) {
	// Decode the base64 encoded signature
	sig, err := base64.StdEncoding.DecodeString(string(signature))
	if err != nil {
		return false, err
	}

	// Verify the signature using VerifyASN1
	valid := ecdsa.VerifyASN1(pubKey, hash, sig)

	return valid, nil
}
