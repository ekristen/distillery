#!/usr/bin/env sh

set -e

if [ -n "${DEBUG:-}" ]; then
  set -x
fi

SKIP_CHECKSUM="${SKIP_CHECKSUM:-}"
RELEASES_URL="https://github.com/ekristen/distillery/releases"
FILE_BASENAME="distillery"
LATEST="v1.7.0-next.6"

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

validate_sha256() {
    local tar_file=$1
    local checksum_file=$2

    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum --ignore-missing --quiet --check "$checksum_file" > /dev/null 2>&1 && return 0
        grep "${tar_file}$" "$checksum_file" > shasum.txt
        sha256sum -c shasum.txt --status > /dev/null 2>&1 && return 0
        sha256sum -c -s shasum.txt > /dev/null 2>&1 && return 0
    fi

    if command -v shasum >/dev/null 2>&1; then
        shasum --ignore-missing -a 256 -c "$checksum_file" > /dev/null 2>&1 && return 0
        shasum -c shasum.txt --status > /dev/null 2>&1 && return 0
    fi

    echo "Unable to verify checksums." >&2
    return 1
}

(
	cd "$TMP_DIR"
	echo "Downloading distillery $VERSION..."
	if ! curl -sfLO "$RELEASES_URL/download/$VERSION/$TAR_FILE"; then
	      echo "Failed to download distillery $VERSION." >&2
    exit 1
  fi
  if ! curl -sfLO "$RELEASES_URL/download/$VERSION/checksums.txt"; then
      echo "Failed to download checksums for distillery $VERSION." >&2
      exit 1
  fi
	echo "Verifying checksums..."
	if validate_sha256 "$TAR_FILE" checksums.txt; then
	  echo "Checksum verification succeeded."
	else
	  if [ -z "$SKIP_CHECKSUM" ]; then
      echo "Checksum verification failed."
      exit 1
    fi
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
"$TMP_DIR/dist" "install" "github/ekristen/distillery@${VERSION}" "$@"
