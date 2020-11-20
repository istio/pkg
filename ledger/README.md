# Legend Package Overview
The legend package exposes the Legend interface, a key value store which provides a unique and reproducible version identifier for every combination of keys and values it has seen.  This history is stored efficiently, using a byte cache, a sparse merkle tree, and a few helper lists and maps.  By default, history is retained indefinitely, but unnecessary versions can be removed from the history using Erase().  Use of the merkle tree for efficient storage also means that keys must be exactly 8 bytes in length, and mumur64 hashing is used to ensure this constraint, meaning keys cannot be retrieved from the trie, only their hashes.  If keys need to be reconstructed, a helper map can track the input key to it's hash.

## Sparse Merkle Trees
The details of this algorithm are documented [here](https://eprint.iacr.org/2016/683.pdf), but a brief overview of Merkle Trees is in order before we consider the specifics of this SMT implementation.  A merkle tree is a key value store represented as a binary tree where the location of a value is determined by traversing downward through the tree, moving right for every set bit of the key, and left for every unset bit.  Thus the value for key 11010 would be found at root.right().right().left().right().left().  Additionally, the value of intermediate nodes is the hash of the node's children, meaning that every change to the tree updates the root node value in logarithmic time.  To avoid the problem of nodes which could be values or could be intermediate nodes, merkle trees require that all keys be of the same length, and values are stored only at the bottom of the tree, in leaf nodes.

Sparse Merkle Trees improve upon this design by allowing shortcuts.  Nodes which have only one value below them are marked as shortcut nodes (in our case by setting byte[8]=1), and their value is stored in their right child node, allowing trees to have values at lower heights to save space.  Shortcut nodes store their keys in the left child node, to prevent key collisions for similar keys which could be at the same shortcut.

In our implementation, nodes are grouped into pages for persistence into the ByteCache, which helps with efficiency.  Each page contains up to 31 nodes, representing four layers of the tree, and the values of the bottom layer of the page are they keys that can be used to retrieve the next page from the ByteCache.  Because page leaves are the implicit root of the subsequent page, the page root node value includes only the shortcut indicator bit, and the left and right children of a leaf node are stored at index 1 and 2 of the next page, respectively.  Effectively, leaf nodes and root nodes overlap.

To ease visualizing and troubleshooting these trees, the DumpToDOT function returns a graphviz representation of the tree, complete with legend.  The diagram differentiates between shortcut keys, shortcut values, and page borders for help in diagnosing issues.  A small tree diagram is included below to illustrate the concept.  

![smt diagram](https://github.com/istio/pkg/blob/master/legend/diagram.svg?raw=true)

## ByteCache
As the tree is updated, new pages are written to the byte cache, but old pages are not deleted or overwritten.  This allows us to retrieve any past version of the tree using only its root node value.  The history is essentially immutable, unless Erase is called...

## Helpers
The ledger has a few helpers which add functionality.  Most notably, the history class contains a linked list of every version of the ledger tree, with a map for easy retrieval of a specific version from the list.  The history class enables the ledger to erase a version from history when it is no longer needed, removing only those pages which are not used in other tracked versions of the tree.

Additionally, the keymap is a map from the hashed map used in the tree to the actual key value supplied by the caller, so that we can reconstruct the entire tree when needed. 
