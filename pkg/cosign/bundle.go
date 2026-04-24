package cosign

import "strings"

type Payload struct {
	Body           string `json:"body"`
	IntegratedTime int64  `json:"integratedTime"`
	LogIndex       int64  `json:"logIndex"`
	LogID          string `json:"logID"`
}

type Rekor struct {
	SignedEntryTimestamp string  `json:"SignedEntryTimestamp"`
	Payload              Payload `json:"Payload"`
}

// Bundle is the legacy cosign bundle format (produced with cosign sign-blob
// --bundle). The certificate is PEM-encoded and the signature is base64.
type Bundle struct {
	Signature   string `json:"base64Signature"`
	Certificate string `json:"cert"`
	RekorBundle Rekor  `json:"rekorBundle"`
}

// SigstoreBundleMediaTypePrefix is the media type prefix for Sigstore
// Protobuf Bundle format (v0.1/v0.2/v0.3).
const SigstoreBundleMediaTypePrefix = "application/vnd.dev.sigstore.bundle"

// SigstoreBundle is the new Sigstore Protobuf Bundle format
// (application/vnd.dev.sigstore.bundle+json). It bundles the verification
// material (certificate or public key) with the message signature and
// optional transparency log entries.
//
// Spec: https://github.com/sigstore/protobuf-specs
type SigstoreBundle struct {
	MediaType            string                       `json:"mediaType"`
	VerificationMaterial SigstoreVerificationMaterial `json:"verificationMaterial"`
	MessageSignature     *SigstoreMessageSignature    `json:"messageSignature,omitempty"`
}

type SigstoreVerificationMaterial struct {
	// Certificate is the leaf signing certificate (raw DER, base64 encoded in JSON).
	Certificate *SigstoreX509Certificate `json:"certificate,omitempty"`
	// PublicKey references a public key by hint (used for keyed signing).
	PublicKey *SigstorePublicKeyIdentifier `json:"publicKey,omitempty"`
	// X509CertificateChain is the legacy v0.1 location for the signing cert
	// (kept for compatibility with bundles emitted by older tooling).
	X509CertificateChain *SigstoreX509CertificateChain `json:"x509CertificateChain,omitempty"`
}

type SigstoreX509CertificateChain struct {
	Certificates []SigstoreX509Certificate `json:"certificates,omitempty"`
}

type SigstoreX509Certificate struct {
	RawBytes string `json:"rawBytes"`
}

type SigstorePublicKeyIdentifier struct {
	Hint string `json:"hint"`
}

type SigstoreMessageSignature struct {
	MessageDigest *SigstoreMessageDigest `json:"messageDigest,omitempty"`
	Signature     string                 `json:"signature"`
}

type SigstoreMessageDigest struct {
	Algorithm string `json:"algorithm"`
	Digest    string `json:"digest"`
}

// IsSigstoreBundle returns true when the bundle's media type identifies it
// as a Sigstore Protobuf Bundle.
func (b *SigstoreBundle) IsSigstoreBundle() bool {
	return b != nil && strings.HasPrefix(b.MediaType, SigstoreBundleMediaTypePrefix)
}

// LeafCertificate returns the raw (base64 DER) leaf signing certificate
// from either the v0.3+ `certificate` field or the legacy v0.1
// `x509CertificateChain.certificates[0]` location.
func (b *SigstoreBundle) LeafCertificate() string {
	if b == nil {
		return ""
	}
	if b.VerificationMaterial.Certificate != nil && b.VerificationMaterial.Certificate.RawBytes != "" {
		return b.VerificationMaterial.Certificate.RawBytes
	}
	if b.VerificationMaterial.X509CertificateChain != nil &&
		len(b.VerificationMaterial.X509CertificateChain.Certificates) > 0 {
		return b.VerificationMaterial.X509CertificateChain.Certificates[0].RawBytes
	}
	return ""
}
