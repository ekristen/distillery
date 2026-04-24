package cosign_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ekristen/distillery/pkg/cosign"
)

func TestParsePublicKey(t *testing.T) {
	// Generate a test ECDSA key pair
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate ECDSA key: %v", err)
	}
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	if err != nil {
		t.Fatalf("Failed to marshal public key: %v", err)
	}
	pubKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	})

	// Test parsing the public key
	parsedPubKey, err := cosign.ParsePublicKey(pubKeyPEM)
	if err != nil {
		t.Fatalf("Failed to parse public key: %v", err)
	}
	if !parsedPubKey.Equal(&privKey.PublicKey) {
		t.Fatalf("Parsed public key does not match original")
	}
}

func TestVerifySignature(t *testing.T) {
	// Generate a test ECDSA key pair
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate ECDSA key: %v", err)
	}

	// Create test data and sign it
	data := []byte("test data")
	hasher := sha256.New()
	hasher.Write(data)
	hash := hasher.Sum(nil)

	sig, err := ecdsa.SignASN1(rand.Reader, privKey, hash)
	if err != nil {
		t.Fatalf("Failed to sign data: %v", err)
	}

	// Encode the signature in base64
	signatureBase64 := base64.StdEncoding.EncodeToString(sig)
	fmt.Println("Signature:", signatureBase64)

	// Test verifying the signature
	valid, err := cosign.VerifySignature(&privKey.PublicKey, hash, []byte(signatureBase64))
	if err != nil {
		t.Fatalf("Failed to verify signature: %v", err)
	}
	if !valid {
		t.Fatalf("Signature verification failed")
	}
}

func TestVerifyChecksumSignature(t *testing.T) {
	// Read the contents of checksums.txt.pem
	publicKeyContentEncoded, err := os.ReadFile("testdata/checksums.txt.pem")
	if err != nil {
		t.Fatalf("Failed to read public key file: %v", err)
	}

	// Decode the base64-encoded public key
	publicKeyContent, err := base64.StdEncoding.DecodeString(string(publicKeyContentEncoded))
	if err != nil {
		t.Fatalf("Failed to decode base64 public key: %v", err)
	}

	// Read the contents of checksums.txt.sig
	signatureContent, err := os.ReadFile("testdata/checksums.txt.sig")
	if err != nil {
		t.Fatalf("Failed to read signature file: %v", err)
	}

	// Read the contents of checksums.txt
	dataContent, err := os.ReadFile("testdata/checksums.txt")
	if err != nil {
		t.Fatalf("Failed to read data file: %v", err)
	}

	// Decode the PEM-encoded public key
	pubKey, err := cosign.ParsePublicKey(publicKeyContent)
	if err != nil {
		t.Fatalf("Failed to parse public key: %v", err)
	}

	dataHash := cosign.HashData(dataContent)

	// Verify the signature
	valid, err := cosign.VerifySignature(pubKey, dataHash, signatureContent)
	if err != nil {
		t.Fatalf("Failed to verify signature: %v", err)
	}
	if !valid {
		t.Fatalf("Signature verification failed")
	}
}

func TestVerifyChecksumSignaturePublicKey(t *testing.T) {
	// Read the contents of checksums.txt.pem
	publicKeyContent, err := os.ReadFile("testdata/release.pub")
	if err != nil {
		t.Fatalf("Failed to read public key file: %v", err)
	}

	// Read the contents of checksums.txt.sig
	signatureContent, err := os.ReadFile("testdata/release.sig")
	if err != nil {
		t.Fatalf("Failed to read signature file: %v", err)
	}

	// Decode the PEM-encoded public key
	pubKey, err := cosign.ParsePublicKey(publicKeyContent)
	if err != nil {
		t.Fatalf("Failed to parse public key: %v", err)
	}

	dataHashEncoded, err := os.ReadFile("testdata/release.sha256")
	if err != nil {
		t.Fatalf("Failed to read data file: %v", err)
	}

	dataHash, err := base64.StdEncoding.DecodeString(string(dataHashEncoded))
	if err != nil {
		t.Fatalf("Failed to decode base64 data hash: %v", err)
	}

	// Verify the signature
	valid, err := cosign.VerifySignature(pubKey, dataHash, signatureContent)
	if err != nil {
		t.Fatalf("Failed to verify signature: %v", err)
	}
	if !valid {
		t.Fatalf("Signature verification failed")
	}
}

// buildSelfSignedCertDER produces a DER-encoded self-signed X.509 cert
// whose public key matches the given ECDSA private key. Used to exercise
// Sigstore bundle parsing without a real Sigstore cert chain.
func buildSelfSignedCertDER(t *testing.T, priv *ecdsa.PrivateKey) []byte {
	t.Helper()
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("failed to create self-signed cert: %v", err)
	}
	return der
}

func TestParseCertificateDER(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	der := buildSelfSignedCertDER(t, priv)

	pub, err := cosign.ParseCertificateDER(der)
	if err != nil {
		t.Fatalf("ParseCertificateDER failed: %v", err)
	}
	if !pub.Equal(&priv.PublicKey) {
		t.Fatalf("public key mismatch")
	}
}

func TestSigstoreBundleHelpers(t *testing.T) {
	t.Run("detects sigstore media type", func(t *testing.T) {
		b := &cosign.SigstoreBundle{MediaType: "application/vnd.dev.sigstore.bundle.v0.3+json"}
		if !b.IsSigstoreBundle() {
			t.Fatalf("expected sigstore bundle to be detected")
		}
	})

	t.Run("rejects non-sigstore media type", func(t *testing.T) {
		b := &cosign.SigstoreBundle{MediaType: "application/json"}
		if b.IsSigstoreBundle() {
			t.Fatalf("did not expect plain json to register as sigstore")
		}
	})

	t.Run("leaf cert prefers verificationMaterial.certificate", func(t *testing.T) {
		b := &cosign.SigstoreBundle{
			VerificationMaterial: cosign.SigstoreVerificationMaterial{
				Certificate: &cosign.SigstoreX509Certificate{RawBytes: "NEW"},
				X509CertificateChain: &cosign.SigstoreX509CertificateChain{
					Certificates: []cosign.SigstoreX509Certificate{{RawBytes: "OLD"}},
				},
			},
		}
		if got := b.LeafCertificate(); got != "NEW" {
			t.Fatalf("expected NEW cert, got %q", got)
		}
	})

	t.Run("leaf cert falls back to legacy chain", func(t *testing.T) {
		b := &cosign.SigstoreBundle{
			VerificationMaterial: cosign.SigstoreVerificationMaterial{
				X509CertificateChain: &cosign.SigstoreX509CertificateChain{
					Certificates: []cosign.SigstoreX509Certificate{{RawBytes: "OLD"}},
				},
			},
		}
		if got := b.LeafCertificate(); got != "OLD" {
			t.Fatalf("expected OLD cert, got %q", got)
		}
	})
}

// TestSigstoreBundleUnmarshal ensures the JSON field names used by cosign
// 3.x for the application/vnd.dev.sigstore.bundle+json format map onto
// our struct definitions.
func TestSigstoreBundleUnmarshal(t *testing.T) {
	raw := []byte(`{
		"mediaType": "application/vnd.dev.sigstore.bundle.v0.3+json",
		"verificationMaterial": {
			"certificate": {"rawBytes": "Y2VydA=="}
		},
		"messageSignature": {
			"messageDigest": {"algorithm": "SHA2_256", "digest": "ZGln"},
			"signature": "c2ln"
		}
	}`)

	var b cosign.SigstoreBundle
	if err := json.Unmarshal(raw, &b); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if !b.IsSigstoreBundle() {
		t.Fatalf("expected sigstore bundle")
	}
	if b.LeafCertificate() != "Y2VydA==" {
		t.Fatalf("unexpected cert: %s", b.LeafCertificate())
	}
	if b.MessageSignature == nil || b.MessageSignature.Signature != "c2ln" {
		t.Fatalf("unexpected signature: %+v", b.MessageSignature)
	}
	if b.MessageSignature.MessageDigest.Algorithm != "SHA2_256" {
		t.Fatalf("unexpected algorithm: %s", b.MessageSignature.MessageDigest.Algorithm)
	}
}

// TestSigstoreBundleEndToEnd verifies the ECDSA signature contained in a
// self-constructed Sigstore bundle against hashed content — covering the
// full parse → pubkey-extract → VerifySignature path the provider uses.
func TestSigstoreBundleEndToEnd(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("key gen: %v", err)
	}
	certDER := buildSelfSignedCertDER(t, priv)

	message := []byte("payload-to-sign")
	hash := sha256.Sum256(message)
	sigDER, err := ecdsa.SignASN1(rand.Reader, priv, hash[:])
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	bundle := cosign.SigstoreBundle{
		MediaType: "application/vnd.dev.sigstore.bundle.v0.3+json",
		VerificationMaterial: cosign.SigstoreVerificationMaterial{
			Certificate: &cosign.SigstoreX509Certificate{
				RawBytes: base64.StdEncoding.EncodeToString(certDER),
			},
		},
		MessageSignature: &cosign.SigstoreMessageSignature{
			MessageDigest: &cosign.SigstoreMessageDigest{
				Algorithm: "SHA2_256",
				Digest:    base64.StdEncoding.EncodeToString(hash[:]),
			},
			Signature: base64.StdEncoding.EncodeToString(sigDER),
		},
	}

	raw, err := json.Marshal(&bundle)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var parsed cosign.SigstoreBundle
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	leafDER, err := base64.StdEncoding.DecodeString(parsed.LeafCertificate())
	if err != nil {
		t.Fatalf("decode cert: %v", err)
	}
	pub, err := cosign.ParseCertificateDER(leafDER)
	if err != nil {
		t.Fatalf("parse cert: %v", err)
	}

	valid, err := cosign.VerifySignature(pub, cosign.HashData(message), []byte(parsed.MessageSignature.Signature))
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !valid {
		t.Fatalf("signature did not verify")
	}
}
