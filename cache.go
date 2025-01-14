package main

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"os"
	"path"

	"github.com/dgraph-io/ristretto/v2"
)

type remoteImageCache interface {
	set(url string, res imageResult) error
	get(url string) (imageResult, error)
}

var errCacheMiss = fmt.Errorf("cache miss")

type noopCache struct{}

func (*noopCache) set(url string, res imageResult) error {
	return nil
}
func (*noopCache) get(url string) (imageResult, error) {
	return imageResult{}, errCacheMiss
}

type memoryCache struct {
	cacheClient *ristretto.Cache[string, imageResult]
}

func newMemoryCache() *memoryCache {
	c, err := ristretto.NewCache(&ristretto.Config[string, imageResult]{
		NumCounters: 10000,
		MaxCost:     100000000,
		BufferItems: 64,
		Cost: func(value imageResult) int64 {
			return int64(len(value.data))
		},
		ShouldUpdate: func(cur, prev imageResult) bool {
			return len(cur.data) != len(prev.data)
		},
	})
	if err != nil {
		panic(err)
	}
	return &memoryCache{
		cacheClient: c,
	}
}
func (c *memoryCache) set(url string, res imageResult) error {
	c.cacheClient.Set(url, res, 0)
	return nil
}
func (c *memoryCache) get(url string) (imageResult, error) {
	res, hasRes := c.cacheClient.Get(url)
	if !hasRes {
		return imageResult{}, errCacheMiss
	}
	return res, nil
}

type fileCache struct {
	basePath string
}

func (c *fileCache) set(url string, res imageResult) error {
	cacheBytes := make([]byte, binary.MaxVarintLen64+len(res.data))
	binary.PutVarint(cacheBytes, int64(res.format))
	copy(cacheBytes[binary.MaxVarintLen64:], res.data)
	return os.WriteFile(c.cachePath(url), cacheBytes, os.ModePerm)
}
func (c *fileCache) get(url string) (imageResult, error) {
	cachePath := c.cachePath(url)
	if _, err := os.Stat(cachePath); err != nil {
		return imageResult{}, errCacheMiss
	}
	cachebytes, err := os.ReadFile(cachePath)
	if err != nil {
		return imageResult{}, fmt.Errorf("error reading cache: %w", err)
	}
	format, err := binary.ReadVarint(bytes.NewBuffer(cachebytes))
	if err != nil {
		return imageResult{}, fmt.Errorf("invalid cache entry: unable to read format: %w", err)
	}
	return imageResult{nil, cachebytes[binary.MaxVarintLen64:], imageFormat(format)}, nil
}
func (c *fileCache) cachePath(url string) string {
	hash := md5.Sum([]byte(url))
	return path.Join(c.basePath, fmt.Sprintf("%x.cache", hash))
}
