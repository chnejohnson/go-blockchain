package blockchain

import "crypto/sha256"

// MerkleTree contains the root node of the merkle tree
type MerkleTree struct {
	RootNode *MerkleNode
}

// MerkleNode has the data and the recursive tree structure linked another node
type MerkleNode struct {
	Left  *MerkleNode
	Right *MerkleNode
	Data  []byte
}

// NewMerkleNode creates a new node from left node and right node data
func NewMerkleNode(left, right *MerkleNode, data []byte) *MerkleNode {
	node := MerkleNode{}

	if left == nil && right == nil {
		hash := sha256.Sum256(data)
		node.Data = hash[:]
	} else {
		prevHashes := append(left.Data, right.Data...)
		hash := sha256.Sum256(prevHashes)
		node.Data = hash[:]
	}

	node.Left = left
	node.Right = right

	return &node
}

// NewMerkleTree creates the merkle tree by slice of hash of transaction
func NewMerkleTree(data [][]byte) *MerkleTree {
	var nodes []MerkleNode

	// 如果交易加總不是偶數，就要複製最後一個交易，讓總量變成偶數
	if len(data)%2 != 0 {
		data = append(data, data[len(data)-1])
	}

	// 製作出最底層的 nodes
	for _, d := range data {
		node := NewMerkleNode(nil, nil, d)
		nodes = append(nodes, *node)
	}

	// 製作最後一個 level 的 node，也就是 root node
	for i := 0; i < len(data)/2; i++ {
		var level []MerkleNode

		for j := 0; j < len(nodes); j += 2 {
			node := NewMerkleNode(&nodes[j], &nodes[j+1], nil)
			level = append(level, *node)
		}

		nodes = level
	}

	// 建立 MerkleTree
	tree := MerkleTree{&nodes[0]}

	return &tree
}
