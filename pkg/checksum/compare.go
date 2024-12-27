package checksum

import (
	"bufio"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

func ComputeFileHash(filePath string, hashFunc func() hash.Hash) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	h := hashFunc()
	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// CompareHashWithChecksumFile compares the computed hash of a file with the hashes in a checksum file.
func CompareHashWithChecksumFile(srcFilename, srcFilePath, checksumFilePath string, hashFunc func() hash.Hash) (bool, error) {
	log := logrus.WithField("handler", "compare-hash-with-checksum-file")

	// Compute the hash of the file
	computedHash, err := ComputeFileHash(srcFilePath, hashFunc)
	if err != nil {
		return false, err
	}

	// Open the checksum file
	checksumFile, err := os.Open(checksumFilePath)
	if err != nil {
		return false, err
	}
	defer checksumFile.Close()

	// Read and compare hashes
	scanner := bufio.NewScanner(checksumFile)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)

		var fileHash string
		var hashFilename string

		if len(parts) > 1 {
			fileHash = parts[0]
			hashFilename = parts[1]
		} else if len(parts) > 0 {
			fileHash = parts[0]
			hashFilename = srcFilename
		} else {
			return false, fmt.Errorf("unable to find hash and filename in checksum file")
		}

		log.Trace("fileHash: ", fileHash)
		log.Trace("filename: ", hashFilename)
		// Rust does *(binary) for the binary name
		hashFilename = strings.TrimPrefix(hashFilename, "*")

		if (hashFilename == srcFilename || filepath.Base(hashFilename) == srcFilename) && fileHash == computedHash {
			return true, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return false, err
	}

	return false, nil
}
