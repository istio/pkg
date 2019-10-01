package ledger

import (
	"fmt"
	"strings"
)

type Ledger interface {
	Put(key, value string) (string, error)
	Delete(key string) error
	Get(key string) (string, error)
	RootHash() string
	GetPreviousValue(previousHash, key string)(result string, err error)
}

type SMTLedger struct {
	t SMT
}

func (s *SMTLedger) Put(key, value string) (result string, err error) {
	b, err := s.t.AtomicUpdate([][]byte{[]byte(fmt.Sprintf("%8v", key))}, [][]byte{[]byte(fmt.Sprintf("%8v", value))})
	//b, err := s.t.AtomicUpdate([][]byte{trie.Hasher([]byte(key))}, [][]byte{[]byte(fmt.Sprintf("%8v", value))})
	result = string(b)
	return
}

func (s *SMTLedger) Delete(key string) (err error) {
	_, err = s.t.Update([][]byte{[]byte(key)}, [][]byte{DefaultLeaf})
	return
}

func (s *SMTLedger) GetPreviousValue(previousHash, key string)(result string, err error) {
	b, err := s.t.GetPreviousValue([]byte(previousHash), []byte(fmt.Sprintf("%8v", key)))
	result = strings.TrimSpace(string(b))
	return
}

func (s *SMTLedger) Get(key string) (result string, err error) {
	return s.GetPreviousValue(s.RootHash(), key)
}

func (s *SMTLedger) RootHash() string {
	return string(s.t.Root)
}

