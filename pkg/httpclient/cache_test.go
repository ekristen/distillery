package httpclient

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDiskCache_WritesWithRestrictivePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "test-cache")

	cache := NewDiskCache(cachePath)

	// Write a cache entry
	cache.Set("test-key", []byte("test-value"))

	// Verify the value is retrievable
	val, ok := cache.Get("test-key")
	assert.True(t, ok)
	assert.Equal(t, []byte("test-value"), val)

	// Check the cache directory permissions
	info, err := os.Stat(cachePath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0700), info.Mode().Perm(),
		"cache directory should have 0700 permissions")

	// Find and check the cache file permissions
	entries, err := os.ReadDir(cachePath)
	require.NoError(t, err)
	require.NotEmpty(t, entries, "cache directory should contain at least one file")

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		fileInfo, err := entry.Info()
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0600), fileInfo.Mode().Perm(),
			"cache file %s should have 0600 permissions", entry.Name())
	}
}

func TestNewDiskCache_DeleteRemovesEntry(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "test-cache")

	cache := NewDiskCache(cachePath)

	cache.Set("ephemeral", []byte("should-be-deleted"))
	val, ok := cache.Get("ephemeral")
	assert.True(t, ok)
	assert.Equal(t, []byte("should-be-deleted"), val)

	cache.Delete("ephemeral")
	_, ok = cache.Get("ephemeral")
	assert.False(t, ok, "deleted entry should not be retrievable")
}
