package ledger

import (
	"container/list"
	"sync"
)

type history struct {
	*list.List
	index map[string]*list.Element

	// lock is for the whole struct
	lock sync.RWMutex
}

func NewHistory() history {
	return history{
		List:  list.New(),
		index: make(map[string]*list.Element),
	}
}

func (h *history) Get(hash string) *list.Element {
	return h.index[hash]
}

func (h *history) Put(hash string) *list.Element {
	h.lock.Lock()
	defer h.lock.Unlock()
	result := h.PushBack(hash)
	h.index[hash] = result
	return result
}