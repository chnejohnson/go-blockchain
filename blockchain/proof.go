package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"math/big"
)

// Take the data from the blockchain
// create a counter (nonce) which starts at 0
// create a hash of the data plus the counter
// check the hash to see if it meets a set of requirements

// Requirements:
// The first new bytes must contain 0s

// 困難產生區塊，快速簡單驗證

// Difficulty is the difficulty for miner to mine
const Difficulty = 16

// ProofOfWork is a struct binding Block and Target
type ProofOfWork struct {
	Block  *Block
	Target *big.Int
}

// NewProof create a binding btw Block and Target,
// Target is a big integer
func NewProof(b *Block) *ProofOfWork {
	target := big.NewInt(1)
	target.Lsh(target, uint(256-Difficulty))
	pow := &ProofOfWork{b, target}
	return pow
}

// Run select the nonce and the hash to assign the Block.
// Using the nonce and the hash to create new Block in CreateBlock function.
func (pow *ProofOfWork) Run() (int, []byte) {
	var intHash big.Int
	var hash [32]byte

	nonce := 0

	for nonce < math.MaxInt64 {
		data := pow.InitData(nonce)
		hash = sha256.Sum256(data)

		// 計算本身很快，是Atom在Printf的視界中刻意放慢的
		fmt.Printf("\r%x -- %d", hash, nonce)

		// turn byte slice into big.Int
		intHash.SetBytes(hash[:])

		// 產生一個 hash 去跟某一個 block 綁定的 target 比大小
		if intHash.Cmp(pow.Target) == -1 {
			// intHash < pow.Target
			break
		} else {
			// intHash >= pow.Target
			nonce++
		}
	}

	fmt.Println()
	return nonce, hash[:]
}

// InitData create byte slice with some of the resource
func (pow *ProofOfWork) InitData(nonce int) []byte {
	data := bytes.Join(
		[][]byte{
			pow.Block.HashTransactions(),
			pow.Block.PrevHash,
			ToHex(int64(nonce)),
			ToHex(int64(Difficulty)),
		}, []byte{})

	return data
}

// Validate confirm hash is less than target
func (pow *ProofOfWork) Validate() bool {
	var intHash big.Int

	data := pow.InitData(pow.Block.Nonce)

	hash := sha256.Sum256(data)
	intHash.SetBytes(hash[:])

	return intHash.Cmp(pow.Target) == -1
}

// ToHex turn int64 into byte slice
func ToHex(num int64) []byte {
	buff := new(bytes.Buffer)
	err := binary.Write(buff, binary.BigEndian, num)
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}
