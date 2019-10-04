/**
 *  @file
 *  @copyright defined in aergo/LICENSE.txt
 */

package ledger

import (
	"bytes"
	"github.com/spaolacci/murmur3"
)

var (
	// Trie default value : hash of 0x0
	DefaultLeaf = Hasher([]byte{0x0})
)

const (
	HashLength   = 8
)

type Hash [HashLength]byte

func bitIsSet(bits []byte, i int) bool {
	return bits[i/8]&(1<<uint(7-i%8)) != 0
}
func bitSet(bits []byte, i int) {
	bits[i/8] |= 1 << uint(7-i%8)
}

func Hasher(data ...[]byte) []byte {
	var hasher = murmur3.New64()
	for i := 0; i < len(data); i++ {
		hasher.Write(data[i])
	}
	result := hasher.Sum(nil)
	return result
}
// for sorting
type DataArray [][]byte

func (d DataArray) Len() int {
	return len(d)
}
func (d DataArray) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}
func (d DataArray) Less(i, j int) bool {
	return bytes.Compare(d[i], d[j]) == -1
}
