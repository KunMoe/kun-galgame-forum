package markdown

import "sync"

var stickers = struct {
	sync.RWMutex
	hashes map[string]struct{}
}{hashes: seedStickerHashes()}

func seedStickerHashes() map[string]struct{} {
	out := make(map[string]struct{}, len(legacyStickerHashByKey))
	for _, h := range legacyStickerHashByKey {
		out[h] = struct{}{}
	}
	return out
}

func IsSticker(hash string) bool {
	stickers.RLock()
	defer stickers.RUnlock()
	_, ok := stickers.hashes[hash]
	return ok
}

func LearnStickers(hashes []string) {
	stickers.Lock()
	defer stickers.Unlock()
	for _, h := range hashes {
		stickers.hashes[h] = struct{}{}
	}
}
