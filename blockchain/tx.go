package blockchain

import (
	"bytes"

	"github.com/go-blockchain/wallet"
)

// TxInput is just a reference to previous output
type TxInput struct {
	ID        []byte
	Out       int
	Signature []byte
	PubKey    []byte
}

// TxOutput has Value which is the transaction token,
// and the Pubkey is for unlocking the Value field.
// In bitcoin, the Pubkey is derived from a complicated scripting language called 'script'
// we just use arbitrary key to represent the user's address
type TxOutput struct {
	Value      int    // how much token be send
	PubKeyHash []byte // the token receiver's address
}

// NewTXOutput create a new TxOutput with value and address
func NewTXOutput(value int, address string) *TxOutput {
	txo := &TxOutput{value, nil}
	txo.Lock([]byte(address))
	return txo
}

// UsesKey check output's pubKeyHash is the same as input public key
func (in *TxInput) UsesKey(pubKeyHash []byte) bool {
	// hashing the input public key
	lockingHash := wallet.PublicKeyHash(in.PubKey)

	return bytes.Compare(lockingHash, pubKeyHash) == 0
}

// Lock get address's public key, and assign into output's PubKeyHash
func (out *TxOutput) Lock(address []byte) {
	pubKeyHash := wallet.Base58Decode(address)
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	out.PubKeyHash = pubKeyHash
}

// IsLockedWithKey check the pubKeyHash is equal to output's PubKeyHash so that you can unlock output with the key
func (out *TxOutput) IsLockedWithKey(pubKeyHash []byte) bool {
	return bytes.Compare(out.PubKeyHash, pubKeyHash) == 0
}
