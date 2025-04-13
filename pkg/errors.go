package blobcache

import "errors"

var (
	ErrHostNotFound            = errors.New("host not found")
	ErrUnableToReachHost       = errors.New("unable to reach host")
	ErrInvalidHostVersion      = errors.New("invalid host version")
	ErrContentNotFound         = errors.New("content not found")
	ErrClientNotFound          = errors.New("client not found")
	ErrCacheLockHeld           = errors.New("cache lock held")
	ErrUnableToPopulateContent = errors.New("unable to populate content from original source")
	ErrBlobFsMountFailure      = errors.New("failed to mount blobfs")
	ErrUnableToAcquireLock     = errors.New("unable to acquire lock")
)
