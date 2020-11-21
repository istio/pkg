// Copyright 2019 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ledger

import (
	"bytes"
	"fmt"
	"istio.io/pkg/cache"
)

func (s *smt) Root() []byte {
	s.rootMu.RLock()
	defer s.rootMu.RUnlock()
	return s.root
}

// Get fetches the value of a key by going down the current trie root.
func (s *smt) Get(key []byte) ([]byte, error) {
	return s.GetPreviousValue(s.Root(), key)
}

// GetPreviousValue returns the value as of the specified root hash.
func (s *smt) GetPreviousValue(prevRoot []byte, key []byte) ([]byte, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	n, err := buildRootNode(prevRoot, s.trieHeight, s.db)
	if err != nil {
		return nil, err
	}
	return s.get(n, key)
}

// get fetches the value of a key given a trie root
func (s *smt) get(node *node, key []byte) ([]byte, error) {
	if node == nil || len(node.val) == 0 {
		return nil, nil
	}
	height := node.height()
	if height == 0 {
		return node.val[:hashLength], nil
	}
	if node.isShortcut() {
		// shortcuts store their key on left, and value on right
		if bytes.Equal(node.left().val[:hashLength], key) {
			return node.right().val[:hashLength], nil
		}
		return nil, nil
	}
	if bitIsSet(key, s.trieHeight-height) {
		// visit right node
		return s.get(node.right(), key)
	}
	// visit left node
	return s.get(node.left(), key)
}

func (s *smt) GetAll() (keys, values [][]byte, err error) {
	return s.GetAllPrevious(s.root)
}

func (s *smt) Stats() cache.Stats {
	return s.db.Stats()
}

func (s *smt) GetAllPrevious(prevRoot []byte) (keys, values [][]byte, err error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	n, err := buildRootNode(prevRoot, s.trieHeight, s.db)
	if err != nil {
		return nil, nil, err
	}
	return s.getAll(n, make([]byte, hashLength), s.trieHeight)
}

func (s *smt) getAll(n *node, keySoFar []byte, trieHeight byte) (keys, values [][]byte, err error) {
	if n == nil {
		return nil, nil, nil
	} else if n.isShortcut() {
		return [][]byte{n.left().val}, [][]byte{n.right().val}, nil
	} else if n.height() == 0 {
		return [][]byte{keySoFar}, [][]byte{n.val}, nil
	} else {
		lkeys, lvalues, err := s.getAll(n.left(), keySoFar, trieHeight)
		if err != nil {
			return nil, nil, err
		}
		rkey := make([]byte, hashLength)
		copy(rkey, keySoFar)
		setBit(rkey, trieHeight-n.height())
		rkeys, rvalues, err := s.getAll(n.right(), rkey, trieHeight)
		if err != nil {
			return nil, nil, err
		}
		return append(lkeys, rkeys...), append(lvalues, rvalues...), nil
	}
}

func (s *smt) DumpToDOT() string {
	return s.DumpToDOTPrev(s.root)
}

func (s *smt) DumpToDOTPrev(prevRoot []byte) string {
	s.lock.RLock()
	defer s.lock.RUnlock()
	n, err := buildRootNode(prevRoot, s.trieHeight, s.db)
	if err != nil {
		return ""
	}
	inn, _ := s.dumpToDOT(n, 0, 0)
	return fmt.Sprintf("digraph SMT {\nnode [fontname=\"Arial\" style=\"filled\" colorscheme=gnbu3];\n%s\n%s}", legend, inn)
}

func (s *smt) dumpToDOT(n *node, nullCounter int, color int) (string, int) {
	if n == nil {
		s := fmt.Sprintf("null%d;\nnull%d [shape=point];\n", nullCounter, nullCounter)
		nullCounter++
		return s, nullCounter
	}
	var left, right string
	if color == 0 && n.isShortcut() {
		left, nullCounter = s.dumpToDOT(n.left(), nullCounter, 3)
		right, nullCounter = s.dumpToDOT(n.right(), nullCounter, 2)
	} else {
		left, nullCounter = s.dumpToDOT(n.left(), nullCounter, 0)
		right, nullCounter = s.dumpToDOT(n.right(), nullCounter, 0)
	}
	me := fmt.Sprintf("%x", n.val) //[len(n.val)*2-8:]
	result := fmt.Sprintf("\"%s\";\n\"%s\"->%s\"%s\"->%s", me, me, left, me, right)
	if color > 0 {
		// color key and value nodes
		result = fmt.Sprintf("%s\"%s\" [fillcolor=%d]\n", result, me, color)
	}
	if n.isLeaf() && color == 0 {
		// page borders are boxes
		result = fmt.Sprintf("%s\"%s\" [shape=box]\n", result, me)
	}
	return result, nullCounter
}

const legend = `
subgraph legend {
    "shortcut key" [fillcolor=3]
    "shortcut val"  [fillcolor=2]
    "page border" [shape=box]
}
`
