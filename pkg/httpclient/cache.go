package httpclient

import (
	"github.com/gregjones/httpcache/diskcache"
	"github.com/peterbourgon/diskv"
)

// NewDiskCache returns a diskcache.Cache with restrictive file permissions
// (0600 for files, 0700 for directories) to prevent other users from reading
// cached API responses that may contain sensitive data.
func NewDiskCache(basePath string) *diskcache.Cache {
	return diskcache.NewWithDiskv(diskv.New(diskv.Options{
		BasePath:     basePath,
		CacheSizeMax: 100 * 1024 * 1024, // 100MB
		FilePerm:     0600,
		PathPerm:     0700,
	}))
}
