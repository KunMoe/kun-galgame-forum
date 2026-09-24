package client

import (
	"kun-galgame-api/pkg/imageclient"
)

type ImageMetaResolver func(hashes []string) map[string]imageclient.ImageMeta

func (c *GalgameClient) SetImageMetaResolver(resolve ImageMetaResolver) {
	c.imageMeta = resolve
}

type ArtMeta struct {
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Thumbhash string `json:"thumbhash"`
}

func (c *GalgameClient) resolveArtMeta(urls []string) map[string]ArtMeta {
	if c.imageMeta == nil || len(urls) == 0 {
		return nil
	}
	hashByURL := make(map[string]string, len(urls))
	hashes := make([]string, 0, len(urls))
	for _, u := range urls {
		hash := hashFromURL(u)
		if hash == "" {
			continue
		}
		if _, seen := hashByURL[u]; seen {
			continue
		}
		hashByURL[u] = hash
		hashes = append(hashes, hash)
	}
	if len(hashes) == 0 {
		return nil
	}
	metaByHash := c.imageMeta(hashes)
	out := make(map[string]ArtMeta, len(hashByURL))
	for u, hash := range hashByURL {
		if m, ok := metaByHash[hash]; ok {
			out[u] = ArtMeta{Width: m.Width, Height: m.Height, Thumbhash: m.Thumbhash}
		}
	}
	return out
}
