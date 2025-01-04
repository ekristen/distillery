package checksum

import (
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComputeFileHash(t *testing.T) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "testfile-*.txt")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// Write random content to the file
	content := []byte("random content for testing")
	_, err = tmpFile.Write(content)
	assert.NoError(t, err)
	tmpFile.Close()

	// Compute the hash of the file
	hash, err := ComputeFileHash(tmpFile.Name(), sha256.New)
	assert.NoError(t, err)

	// Compute the expected hash
	expectedHash := fmt.Sprintf("%x", sha256.Sum256(content))

	// Assert that the computed hash matches the expected hash
	assert.Equal(t, expectedHash, hash)
}

func TestCompareHashWithChecksumFile(t *testing.T) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "testfile-*.txt")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// Write random content to the file
	content := []byte("random content for testing")
	_, err = tmpFile.Write(content)
	assert.NoError(t, err)
	tmpFile.Close()

	// Compute the hash of the file
	hash := fmt.Sprintf("%x", sha256.Sum256(content))

	// Create a temporary checksum file
	checksumFile, err := os.CreateTemp("", "checksum-*.txt")
	assert.NoError(t, err)
	defer os.Remove(checksumFile.Name())

	// Write the hash and filename to the checksum file
	_, err = fmt.Fprintf(checksumFile, "%s %s\n", hash, tmpFile.Name())
	assert.NoError(t, err)
	checksumFile.Close()

	// Compare the hash with the checksum file
	match, err := CompareHashWithChecksumFile(tmpFile.Name(), tmpFile.Name(), checksumFile.Name())
	assert.NoError(t, err)
	assert.True(t, match)

	// Test with a different hash function
	hash = fmt.Sprintf("%x", sha512.Sum512(content))
	checksumFile, err = os.CreateTemp("", "checksum-*.txt")
	assert.NoError(t, err)
	defer os.Remove(checksumFile.Name())

	_, err = fmt.Fprintf(checksumFile, "%s %s\n", hash, tmpFile.Name())
	assert.NoError(t, err)
	checksumFile.Close()

	match, err = CompareHashWithChecksumFile(tmpFile.Name(), tmpFile.Name(), checksumFile.Name())
	assert.NoError(t, err)
	assert.True(t, match)
}
