package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
)

// Transaction store input and output. Every block has a slice of transaction.
type Transaction struct {
	ID      []byte
	Inputs  []TxInput
	Outputs []TxOutput
}

// TxInput is just a reference to previous output
type TxInput struct {
	ID  []byte // the transaction that output is inside of
	Out int    // the index where this output appear
	Sig string // a script which provide a data used in the output pubkey, such as user account
}

// TxOutput has Value which is the transaction token,
// and the Pubkey is for unlocking the Value field.
// In bitcoin, the Pubkey is derived from a complicated scripting language called 'script'
// we just use arbitrary key to represent the user's address
type TxOutput struct {
	Value  int    // how much token be send
	Pubkey string // the token receiver's address
}

// CoinbaseTx create the first transaction for gensis block
func CoinbaseTx(to, data string) *Transaction {
	if data == "" {
		data = fmt.Sprintf("Coins to %s", to)
	}

	txin := TxInput{[]byte{}, -1, data}
	txout := TxOutput{100, to}

	tx := Transaction{nil, []TxInput{txin}, []TxOutput{txout}}
	tx.SetID()

	return &tx
}

// NewTransaction create a new Transaction for general block
func NewTransaction(from, to string, amount int, chain *Blockchain) *Transaction {
	var inputs []TxInput
	var outputs []TxOutput

	acc, validOutputs := chain.FindSpendableOutputs(from, amount)

	// 檢查是否超過可傳送的金額
	if amount > acc {
		log.Panic("Error: not enough funds")
	}

	// create Input by looping validOutputs (spendable outputs)
	for txid, outs := range validOutputs {
		txID, err := hex.DecodeString(txid)
		Handle(err)

		for _, out := range outs {
			// create Input
			inputs = append(inputs, TxInput{txID, out, from})
		}
	}

	//  create Output
	outputs = append(outputs, TxOutput{amount, to})

	if amount < acc {
		outputs = append(outputs, TxOutput{acc - amount, from})
	}

	tx := Transaction{nil, inputs, outputs}
	tx.SetID()

	return &tx
}

// SetID generate transaction's ID with sha256 in 32 bytes
func (tx *Transaction) SetID() {
	var encoded bytes.Buffer
	var hash [32]byte

	encode := gob.NewEncoder(&encoded)
	err := encode.Encode(tx)
	Handle(err)

	hash = sha256.Sum256(encoded.Bytes())
	tx.ID = hash[:]
}

// IsCoinbase check whether the transaction is coinbase transaction
func (tx *Transaction) IsCoinbase() bool {
	return len(tx.Inputs) == 1 && len(tx.Inputs[0].ID) == 0 && tx.Inputs[0].Out == -1
}

// CanUnlock check if data is input's signature
func (in *TxInput) CanUnlock(data string) bool {
	return in.Sig == data
}

// CanBeUnlocked check if data is output's Pubkey
func (out *TxOutput) CanBeUnlocked(data string) bool {
	return out.Pubkey == data
}
