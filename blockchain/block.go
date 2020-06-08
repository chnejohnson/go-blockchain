package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"log"
)

// Block represent the data of a block
type Block struct {
	Hash         []byte
	Transactions []*Transaction
	PrevHash     []byte
	Nonce        int
}

// HashTransactions hash the block's slice of Transaction
func (b *Block) HashTransactions() []byte {
	var txHashes [][]byte
	var txHash [32]byte

	// txHashes is a slice of all tx.ID in the block
	for _, tx := range b.Transactions {
		txHashes = append(txHashes, tx.ID)
	}

	// txHash is the hash of transaction which is encrypted from txHashes
	txHash = sha256.Sum256(bytes.Join(txHashes, []byte{}))
	return txHash[:]

}

// CreateBlock create a new Block type
func CreateBlock(txs []*Transaction, prevHash []byte) *Block {
	block := &Block{[]byte{}, txs, prevHash, 0}
	// block.DeriveHash()

	pow := NewProof(block)
	nonce, hash := pow.Run()

	block.Hash = hash[:]
	block.Nonce = nonce

	return block
}

// Genesis create the first block
func Genesis(coinbase *Transaction) *Block {
	return CreateBlock([]*Transaction{coinbase}, []byte{})
}

// Serialize turn Block into slice of byte for store in BadgerDB
func (b *Block) Serialize() []byte {
	var res bytes.Buffer
	encoder := gob.NewEncoder(&res)

	err := encoder.Encode(b)
	Handle(err)

	return res.Bytes()
}

// Deserialize turn slice of byte into Block
func Deserialize(data []byte) *Block {
	var block Block
	decoder := gob.NewDecoder(bytes.NewReader(data))

	err := decoder.Decode(&block)
	Handle(err)

	return &block
}

// Handle handle error simply
func Handle(err error) {
	if err != nil {
		log.Panic(err)
	}
}
