# Exit immediately if a command fails
$ErrorActionPreference = "Stop"

$RELEASES_URL = "https://github.com/ekristen/distillery/releases"
$FILE_BASENAME = "distillery"
$LATEST = "__VERSION__"

# Use the provided version or default to the latest
if (-not $env:VERSION) {
    $VERSION = $LATEST
} else {
    $VERSION = $env:VERSION
}

if (-not $VERSION) {
    Write-Error "Unable to get distillery version."
    exit 1
}

# Create a temporary directory
$TMP_DIR = New-TemporaryFile | ForEach-Object { Remove-Item $_ -Force; New-Item -ItemType Directory -Path $_ }
$trap = {
    Remove-Item -Recurse -Force $TMP_DIR
    exit 1
}

# Detect OS and Architecture
$OS = "windows" # Hardcoded for Windows
$ARCH = if ([System.Environment]::Is64BitProcess) { "amd64" } else { "arm64" }

$ZIP_FILE = "${FILE_BASENAME}-${VERSION}-${OS}-${ARCH}.zip"

# Download distillery
Write-Host "Downloading distillery $VERSION..."
Invoke-WebRequest -Uri "$RELEASES_URL/download/$VERSION/$ZIP_FILE" -OutFile "$TMP_DIR\$ZIP_FILE"
Invoke-WebRequest -Uri "$RELEASES_URL/download/$VERSION/checksums.txt" -OutFile "$TMP_DIR\checksums.txt"

# Verify checksums
Write-Host "Verifying checksums..."
$checksums = Get-Content "$TMP_DIR\checksums.txt" | Where-Object { $_ -match $ZIP_FILE }
if (-not $checksums) {
    Write-Error "Checksum not found for $ZIP_FILE"
    exit 1
}
$expected = $checksums.Split(" ")[0]
$actual = (Get-FileHash -Path "$TMP_DIR\$ZIP_FILE" -Algorithm SHA256).Hash
if ($expected -ne $actual) {
    Write-Error "Checksum verification failed!"
    exit 1
}

# Verify signatures if cosign is available
if (Get-Command cosign -ErrorAction SilentlyContinue) {
    Write-Host "Verifying signatures..."
    $REF = "refs/tags/$VERSION"
    try {
        Invoke-WebRequest -Uri "$RELEASES_URL/download/$VERSION/checksums.txt.sigstore" -OutFile "$TMP_DIR\checksums.txt.sigstore"
        & cosign verify-blob `
            --certificate-identity-regexp "https://github.com/ekristen/distillery.*/.github/workflows/.*.yml@$REF" `
            --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' `
            --bundle "$TMP_DIR\checksums.txt.sigstore" `
            "$TMP_DIR\checksums.txt"
    } catch {
        Write-Warning "Signature verification failed, continuing without verification."
    }
} else {
    Write-Warning "Could not verify signatures, cosign is not installed."
}

# Extract tar file
Write-Host "Extracting distillery..."
Expand-Archive -Path "$TMP_DIR\$ZIP_FILE" -DestinationPath $TMP_DIR -Force

# Run the installation command
Write-Host "Installing distillery..."
& "$TMP_DIR\dist" "install" "ekristen/distillery" @args

# Clean up
Remove-Item -Recurse -Force $TMP_DIR
