package ledger

import (
	"container/list"
	"encoding/base64"
	"sync"
)

type history struct {
	*list.List
	index map[string]*list.Element

	// lock is for the whole struct
	lock sync.RWMutex
}

func NewHistory() *history {
	return &history{
		List:  list.New(),
		index: make(map[string]*list.Element),
	}
}

func (h *history) Get(hash string) *list.Element {
	h.lock.RLock()
	defer h.lock.RUnlock()
	return h.index[hash]
}

func (h *history) Put(key []byte) *list.Element {
	h.lock.Lock()
	defer h.lock.Unlock()
	result := h.PushBack(key)
	h.index[base64.StdEncoding.EncodeToString(key)] = result
	return result
}
