package cache

import (
	"io"
	"time"
)

// Interface is a very simple interface to set/get from the cache
type Interface interface {
	io.Closer
	Set(string, []byte, time.Duration) error
	SetAny(string, any, time.Duration) error
	Get(string) ([]byte, error)
}
