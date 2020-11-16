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
	//return s.get(prevRoot, key, nil, 0, 64)
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
