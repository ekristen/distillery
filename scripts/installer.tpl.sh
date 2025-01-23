#!/usr/bin/env sh

set -eu

if [ -n "${DEBUG:-}" ]; then
  set -x
fi

RELEASES_URL="https://github.com/ekristen/distillery/releases"
FILE_BASENAME="distillery"
LATEST="__VERSION__"

test -z "$VERSION" && VERSION="$LATEST"

test -z "$VERSION" && {
	echo "Unable to get distillery version." >&2
	exit 1
}

TMP_DIR="$(mktemp -d)"
# shellcheck disable=SC2064 # intentionally expands here
trap "rm -rf \"$TMP_DIR\"" EXIT INT TERM

OS="$(uname -s | awk '{print tolower($0)}')"
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64) ARCH="amd64" ;; # Normalize x86_64 to amd64
    aarch64) ARCH="arm64" ;; # Normalize aarch64 to arm64
esac
TAR_FILE="${FILE_BASENAME}-${VERSION}-${OS}-${ARCH}.tar.gz"

check_sha_version() {
    local currentver=$1
    local requiredver=$2
    if [ "$(printf '%s\n' "$requiredver" $1 | sort -V | head -n1)" = "$requiredver" ]; then
            return 0
    else
            return 1
    fi
}

(
	cd "$TMP_DIR"
	echo "Downloading distillery $VERSION..."
	curl -sfLO "$RELEASES_URL/download/$VERSION/$TAR_FILE"
	curl -sfLO "$RELEASES_URL/download/$VERSION/checksums.txt"
	echo "Verifying checksums..."
	if command -v sha256sum >/dev/null 2>&1; then
        if check_sha_version "$(sha256sum --version 2>&1| sed '1q' | cut -f 3)" "8.25"; then
            sha256sum --ignore-missing --quiet --check checksums.txt
        else
            grep "${TAR_FILE}$" checksums.txt > shasum.txt
            sha256sum -c shasum.txt --status
        fi
    elif command -v shasum >/dev/null 2>&1; then
        if check_sha_version "$(shasum --version)" "6.0.1"; then
            shasum --ignore-missing -a 256 -c checksums.txt
        else
            grep "${TAR_FILE}$" checksums.txt > shasum.txt
            shasum -c shasum.txt --status
        fi
    else
        echo "Neither sha256sum nor shasum is available to verify checksums." >&2
    fi
	if command -v cosign >/dev/null 2>&1; then
		echo "Verifying signatures..."
		REF="refs/tags/$VERSION"
		if ! cosign verify-blob \
       --certificate-identity-regexp "https://github.com/ekristen/distillery.*/.github/workflows/.*.yml@$REF" \
       --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
       --cert "$RELEASES_URL/download/$VERSION/checksums.txt.pem" \
       --signature "$RELEASES_URL/download/$VERSION/checksums.txt.sig" \
       checksums.txt; then
        echo "Signature verification failed, continuing without verification."
    else
      echo "Signature verification succeeded."
    fi
	else
		echo "Could not verify signatures, cosign is not installed."
	fi
)

tar -xf "$TMP_DIR/$TAR_FILE" -C "$TMP_DIR"
"$TMP_DIR/dist" "install" "github/ekristen/distillery@{$VERSION}" "$@"
