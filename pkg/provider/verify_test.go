package provider

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
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"

	"github.com/ekristen/distillery/pkg/asset"
	"github.com/ekristen/distillery/pkg/common"
	"github.com/ekristen/distillery/pkg/config"
	"github.com/ekristen/distillery/pkg/cosign"
)

// selfSignedCert builds a self-signed X.509 cert bound to the given ECDSA
// key. The return value is the raw DER; callers that need PEM can wrap it.
func selfSignedCert(t *testing.T, priv *ecdsa.PrivateKey) []byte {
	t.Helper()
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}
	return der
}

// writeFile writes content to a temp file inside t.TempDir and returns its
// absolute path.
func writeFile(t *testing.T, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

// testAsset returns a minimal asset.IAsset whose GetFilePath() points at
// path. That's all verify* needs to locate file contents on disk.
func testAsset(name, path string) asset.IAsset {
	a := asset.New(name, name, "linux", "amd64", "1.0.0")
	a.DownloadPath = path
	return a
}

// newTestProvider returns a Provider wired up for verify-path tests, with
// verification enabled and missing-signature/checksum treated as errors so
// skip paths surface as test failures.
func newTestProvider(t *testing.T) *Provider {
	t.Helper()
	return &Provider{
		Options: &Options{
			Config: &config.Config{
				Settings: &config.Settings{
					SignatureMissing: common.Error,
					ChecksumMissing:  common.Error,
				},
			},
			Settings: map[string]interface{}{},
		},
		Logger: log.With().Str("test", "true").Logger(),
	}
}

// signWithSigstoreBundle creates a minimal Sigstore Protobuf Bundle JSON
// that signs content with priv. Content digest cross-check is exercised by
// populating messageDigest.
func signWithSigstoreBundle(t *testing.T, priv *ecdsa.PrivateKey, content []byte) []byte {
	t.Helper()
	hash := sha256.Sum256(content)
	sigDER, err := ecdsa.SignASN1(rand.Reader, priv, hash[:])
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	bundle := cosign.SigstoreBundle{
		MediaType: "application/vnd.dev.sigstore.bundle.v0.3+json",
		VerificationMaterial: cosign.SigstoreVerificationMaterial{
			Certificate: &cosign.SigstoreX509Certificate{
				RawBytes: base64.StdEncoding.EncodeToString(selfSignedCert(t, priv)),
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
	return raw
}

// signWithLegacyBundle creates a legacy cosign --bundle JSON (PEM cert,
// base64 signature).
func signWithLegacyBundle(t *testing.T, priv *ecdsa.PrivateKey, content []byte) []byte {
	t.Helper()
	hash := sha256.Sum256(content)
	sigDER, err := ecdsa.SignASN1(rand.Reader, priv, hash[:])
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: selfSignedCert(t, priv),
	})
	bundle := cosign.Bundle{
		Signature:   base64.StdEncoding.EncodeToString(sigDER),
		Certificate: string(certPEM),
	}
	raw, err := json.Marshal(&bundle)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return raw
}

// TestVerifyCosignSignatureSigstoreBundleFile exercises the new
// `.sigstore`/`.sigstore.json` bundle format signing a binary asset.
func TestVerifyCosignSignatureSigstoreBundleFile(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.NoError(t, err)

	content := []byte("binary payload")
	binaryPath := writeFile(t, "dist-linux-amd64", content)
	sigPath := writeFile(t, "dist-linux-amd64.sigstore", signWithSigstoreBundle(t, priv, content))

	p := newTestProvider(t)
	p.SignatureType = SignatureTypeFile
	p.Binary = testAsset("dist-linux-amd64", binaryPath)
	p.Signature = testAsset("dist-linux-amd64.sigstore", sigPath)

	assert.NoError(t, p.verifyCosignSignature())
}

// TestVerifyCosignSignatureSigstoreBundleChecksum exercises the flow where
// the sigstore bundle signs the checksum manifest (ekristen/cryptkey
// layout).
func TestVerifyCosignSignatureSigstoreBundleChecksum(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.NoError(t, err)

	checksumContent := []byte("abc123  dist-linux-amd64.tar.gz\n")
	checksumPath := writeFile(t, "checksums.txt", checksumContent)
	sigPath := writeFile(t, "checksums.txt.sigstore", signWithSigstoreBundle(t, priv, checksumContent))

	p := newTestProvider(t)
	p.SignatureType = SignatureTypeChecksum
	p.Checksum = testAsset("checksums.txt", checksumPath)
	p.Signature = testAsset("checksums.txt.sigstore", sigPath)

	assert.NoError(t, p.verifyCosignSignature())
}

// TestVerifyCosignSignatureSigstoreBundleTamperedContent ensures the
// messageDigest cross-check catches a content/bundle mismatch before the
// ECDSA verify even runs.
func TestVerifyCosignSignatureSigstoreBundleTamperedContent(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.NoError(t, err)

	original := []byte("original content")
	tampered := []byte("tampered content")

	binaryPath := writeFile(t, "dist-linux-amd64", tampered)
	sigPath := writeFile(t, "dist-linux-amd64.sigstore", signWithSigstoreBundle(t, priv, original))

	p := newTestProvider(t)
	p.SignatureType = SignatureTypeFile
	p.Binary = testAsset("dist-linux-amd64", binaryPath)
	p.Signature = testAsset("dist-linux-amd64.sigstore", sigPath)

	err = p.verifyCosignSignature()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "digest")
}

// TestVerifyCosignSignatureSigstoreBundleIgnoresStrayKey verifies that a
// misclassified Key asset (e.g. cosign 3.x ships `cosign-linux-pivkey-*`
// files the discover step labels as Key) does not prevent recognizing the
// signature as a self-contained sigstore bundle.
func TestVerifyCosignSignatureSigstoreBundleIgnoresStrayKey(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.NoError(t, err)

	content := []byte("binary payload")
	binaryPath := writeFile(t, "dist-linux-amd64", content)
	sigPath := writeFile(t, "dist-linux-amd64.sigstore.json", signWithSigstoreBundle(t, priv, content))
	// Populate a garbage "key" path that should never be read.
	strayKeyPath := writeFile(t, "unrelated.pub", []byte("not a real key"))

	p := newTestProvider(t)
	p.SignatureType = SignatureTypeFile
	p.Binary = testAsset("dist-linux-amd64", binaryPath)
	p.Signature = testAsset("dist-linux-amd64.sigstore.json", sigPath)
	p.Key = testAsset("unrelated.pub", strayKeyPath)

	assert.NoError(t, p.verifyCosignSignature())
}

// TestVerifyCosignSignatureLegacyBundle covers the pre-v3 cosign --bundle
// format path (PEM cert + base64 sig embedded in JSON, no separate key).
func TestVerifyCosignSignatureLegacyBundle(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.NoError(t, err)

	content := []byte("binary payload")
	binaryPath := writeFile(t, "dist-linux-amd64", content)
	sigPath := writeFile(t, "dist-linux-amd64.sig", signWithLegacyBundle(t, priv, content))

	p := newTestProvider(t)
	p.SignatureType = SignatureTypeFile
	p.Binary = testAsset("dist-linux-amd64", binaryPath)
	p.Signature = testAsset("dist-linux-amd64.sig", sigPath)

	assert.NoError(t, p.verifyCosignSignature())
}

// TestVerifyCosignSignatureKeyAndSig covers the classic goreleaser layout:
// a raw base64 signature file alongside a PEM public key.
func TestVerifyCosignSignatureKeyAndSig(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.NoError(t, err)

	content := []byte("binary payload")
	hash := sha256.Sum256(content)
	sigDER, err := ecdsa.SignASN1(rand.Reader, priv, hash[:])
	assert.NoError(t, err)
	sigB64 := []byte(base64.StdEncoding.EncodeToString(sigDER))

	pubDER, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	assert.NoError(t, err)
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})

	binaryPath := writeFile(t, "dist-linux-amd64", content)
	sigPath := writeFile(t, "dist-linux-amd64.sig", sigB64)
	keyPath := writeFile(t, "dist-linux-amd64.pem", pubPEM)

	p := newTestProvider(t)
	p.SignatureType = SignatureTypeFile
	p.Binary = testAsset("dist-linux-amd64", binaryPath)
	p.Signature = testAsset("dist-linux-amd64.sig", sigPath)
	p.Key = testAsset("dist-linux-amd64.pem", keyPath)

	assert.NoError(t, p.verifyCosignSignature())
}

// TestVerifyCosignSignatureTamperedSig confirms a forged signature fails
// verification with a clear error.
func TestVerifyCosignSignatureTamperedSig(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.NoError(t, err)

	content := []byte("binary payload")
	raw := signWithSigstoreBundle(t, priv, content)

	// Flip a byte inside the base64 signature to invalidate it while
	// keeping the structure parseable.
	var bundle cosign.SigstoreBundle
	assert.NoError(t, json.Unmarshal(raw, &bundle))
	sig, err := base64.StdEncoding.DecodeString(bundle.MessageSignature.Signature)
	assert.NoError(t, err)
	sig[len(sig)-1] ^= 0xFF
	bundle.MessageSignature.Signature = base64.StdEncoding.EncodeToString(sig)
	tampered, err := json.Marshal(&bundle)
	assert.NoError(t, err)

	binaryPath := writeFile(t, "dist-linux-amd64", content)
	sigPath := writeFile(t, "dist-linux-amd64.sigstore", tampered)

	p := newTestProvider(t)
	p.SignatureType = SignatureTypeFile
	p.Binary = testAsset("dist-linux-amd64", binaryPath)
	p.Signature = testAsset("dist-linux-amd64.sigstore", sigPath)

	err = p.verifyCosignSignature()
	assert.Error(t, err)
}

// TestVerifyCosignSignatureInvalidBundleFormat exercises the case where a
// non-bundle JSON (or non-JSON) signature file is supplied with no Key:
// verification should skip rather than fail, matching the pre-existing
// legacy-format fallback.
func TestVerifyCosignSignatureSkipsWithoutKeyOrBundle(t *testing.T) {
	binaryPath := writeFile(t, "dist-linux-amd64", []byte("payload"))
	sigPath := writeFile(t, "dist-linux-amd64.sig", []byte("not-json-and-not-parseable"))

	p := newTestProvider(t)
	p.SignatureType = SignatureTypeFile
	p.Binary = testAsset("dist-linux-amd64", binaryPath)
	p.Signature = testAsset("dist-linux-amd64.sig", sigPath)

	// No Key set, no valid bundle — verifier should skip (the pre-existing
	// behavior preserved for legacy releases without companion keys).
	assert.NoError(t, p.verifyCosignSignature())
}

// TestSigstoreBundleUnsupportedDigestAlgorithm guards against silent
// acceptance of a future non-SHA256 algorithm we don't actually verify.
func TestSigstoreBundleUnsupportedDigestAlgorithm(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.NoError(t, err)

	content := []byte("payload")
	raw := signWithSigstoreBundle(t, priv, content)

	var bundle cosign.SigstoreBundle
	assert.NoError(t, json.Unmarshal(raw, &bundle))
	bundle.MessageSignature.MessageDigest.Algorithm = "SHA2_512"
	tampered, err := json.Marshal(&bundle)
	assert.NoError(t, err)

	binaryPath := writeFile(t, "dist-linux-amd64", content)
	sigPath := writeFile(t, "dist-linux-amd64.sigstore", tampered)

	p := newTestProvider(t)
	p.SignatureType = SignatureTypeFile
	p.Binary = testAsset("dist-linux-amd64", binaryPath)
	p.Signature = testAsset("dist-linux-amd64.sigstore", sigPath)

	err = p.verifyCosignSignature()
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "unsupported") || strings.Contains(err.Error(), "algorithm"),
		"expected algorithm error, got: %v", err)
}
