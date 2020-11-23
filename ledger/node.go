// Copyright Istio Authors
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

import "fmt"

type node struct {
	val      []byte
	index    byte
	page     *page // leaves of a page are roots of the next page.  need to map this somehow...
	nextPage *page
}
type page struct {
	root   []byte
	nodes  []*node
	height byte
	// db holds the cache and related locks
	db *cacheDB
}

func (p *page) store() {
	var h hash
	copy(h[:], p.root)
	p.db.updatedMux.Lock()
	p.db.updatedNodes.Set(h, p.getRawNodes())
	p.db.updatedMux.Unlock()
}

func (p *page) getRawNodes() [][]byte {
	result := make([][]byte, len(p.nodes))
	for i, n := range p.nodes {
		if n != nil {
			result[i] = n.val
		}
	}
	return result
}

func (p *page) delete() {
	var h hash
	copy(h[:], p.root)
	p.db.updatedNodes.Delete(h)
}

func buildRootNode(key []byte, trieHeight byte, db *cacheDB) (*node, error) {
	var p1 *page
	if len(key) == 0 {
		// empty key means this is the beginning of the trie
		p1 = buildPage(nil, trieHeight, db, key)
	} else {
		p1 = retrieveOrBuildPage(db, key, trieHeight)
		if p1 == nil {
			return nil, fmt.Errorf("root node [%s] is unknown", key)
		}
		if len(p1.nodes) < 2 {
			return nil, fmt.Errorf("root node [%s] is empty, this should never happen", key)
		}
	}
	p0 := page{
		db:     db,
		height: trieHeight + 4, // this virtual page sits 1 level above the root
	}
	return &node{
		page:     &p0,
		index:    byte(batchLen - 1),
		val:      key,
		nextPage: p1,
	}, nil
}

func buildPage(rawPage [][]byte, height byte, db *cacheDB, key []byte) *page {
	newPage := page{
		db:     db,
		root:   key,
		height: height,
		nodes:  make([]*node, batchLen),
	}
	if len(rawPage) == 0 {
		rawPage = append(rawPage, []byte{0})
	}
	for i, rawNode := range rawPage {
		if len(rawNode) == 0 {
			continue
		}
		newNode := node{
			val:   rawNode,
			index: byte(i),
			page:  &newPage,
		}
		newPage.nodes[i] = &newNode
	}
	return &newPage
}

func retrieveOrBuildPage(db *cacheDB, key []byte, height byte) *page {
	if key == nil {
		return buildPage(nil, height, db, nil)
	}
	var h hash
	copy(h[:], key)
	rawPage, exists := db.updatedNodes.Get(h)
	rawPage = copy2d(rawPage)
	if exists {
		return buildPage(rawPage, height, db, key)
	}
	return nil
}

// this code looks redundant, but this actually is the fastest way to calculate floor(log2(N)) where N <=31
func heightInPage(i byte) byte {
	if i >= 15 {
		return 4
	} else if i >= 7 {
		return 3
	} else if i >= 3 {
		return 2
	} else if i >= 1 {
		return 1
	} else {
		return 0
	}
}

func (n *node) height() byte {
	// this is mathematically correct but computationally expensive
	return n.page.height - heightInPage(n.index)
}

func (n *node) isShortcut() bool {
	if n.height()%4 != 0 {
		return len(n.val) != 0 && n.val[hashLength] == 1
	}
	return n.getNextPage().nodes[0].val[0] == 1
}

// returns true if node is a leaf of the page
func (n *node) isLeaf() bool {
	return n.index+1 >= 1<<batchHeight //this is easier than calculating 2^4
}

func (n *node) getNextPage() *page {
	if n.nextPage == nil {
		n.nextPage = retrieveOrBuildPage(n.page.db, n.val, n.height())
	}
	return n.nextPage
}

func (n *node) left() *node {
	if n == nil {
		return nil
	}
	if n.isLeaf() {
		np := n.getNextPage()
		if np == nil {
			return nil
		}
		return np.nodes[1]
	}
	return n.page.nodes[leftIndex(n.index)]
}

func (n *node) right() *node {
	if n == nil {
		return nil
	}
	if n.isLeaf() {
		np := n.getNextPage()
		if np == nil {
			return nil
		}
		return np.nodes[2]
	}
	return n.page.nodes[rightIndex(n.index)]
}

func leftIndex(i byte) byte {
	result := 2*i + 1
	if result >= byte(batchLen) {
		return 1
	}
	return result
}

func rightIndex(i byte) byte {
	result := 2*i + 2
	if result >= byte(batchLen) {
		return 2
	}
	return result
}

func (n *node) makeShortcut(key []byte, val []byte) {
	// n.left and n.right must be nil
	// mark n as shortcut node
	var p *page
	if n.isLeaf() {
		p = buildPage([][]byte{}, n.page.height-4, n.page.db, []byte{})
		n.nextPage = p
		p.nodes[0].val = []byte{1}
	} else {
		n.val = make([]byte, hashLength)
		n.val = append(n.val, 1)
		p = n.page
	}
	l := node{
		val:   key,
		index: leftIndex(n.index),
		page:  p,
	}
	r := node{
		val:   val,
		index: rightIndex(n.index),
		page:  p,
	}
	p.nodes[l.index] = &l
	p.nodes[r.index] = &r
}

func (n *node) removeShortcut() {
	if n.isLeaf() {
		p := n.getNextPage()
		p.nodes[0].val[0] = 0
		p.nodes[1] = nil
		p.nodes[2] = nil
	} else {
		n.val[hashLength] = 0
		n.page.nodes[leftIndex(n.index)] = nil
		n.page.nodes[rightIndex(n.index)] = nil
	}
}

func (n *node) calculateHash(hasher func(data ...[]byte) []byte, defaultHashes [][]byte) []byte {
	var h []byte
	if (n.left() == nil || len(n.left().val) == 0) && (n.right() == nil || len(n.right().val) == 0) {
		//s.deleteOldNode(oldRoot) //TODO
		return nil
	} else if n.left() == nil || len(n.left().val) == 0 {
		h = hasher(defaultHashes[n.height()-1], n.right().val)
	} else if n.right() == nil || len(n.right().val) == 0 {
		h = hasher(n.left().val[:hashLength], defaultHashes[n.height()-1])
	} else {
		h = hasher(n.left().val[:hashLength], n.right().val)
	}
	var sc byte
	if n.isShortcut() {
		sc = 1
	} else {
		sc = 0
	}
	return append(h, sc)
}

//for leaf shortcut nodes, persists child page
//for top-level nodes, persists own page
func (n *node) store() {
	if n.isLeaf() && n.nextPage != nil {
		n.nextPage.root = n.val
		n.nextPage.store()
		n.nextPage = nil
	}
}

func (n *node) initLeft() {
	if !n.isLeaf() && n.left() == nil {
		i := leftIndex(n.index)
		n.page.nodes[i] = &node{
			page:  n.page,
			index: i,
		}
	} else if n.left() == nil {
		i := leftIndex(n.index)
		p := n.getNextPage()
		p.nodes[i] = &node{
			page:  p,
			index: i,
		}
	}
}

func (n *node) initRight() {
	if !n.isLeaf() && n.right() == nil {
		i := rightIndex(n.index)
		n.page.nodes[i] = &node{
			page:  n.page,
			index: i,
		}
	} else if n.right() == nil {
		i := rightIndex(n.index)
		p := n.getNextPage()
		p.nodes[i] = &node{
			page:  p,
			index: i,
		}
	}
}
