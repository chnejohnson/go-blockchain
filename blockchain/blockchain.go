package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"runtime"

	"github.com/dgraph-io/badger"
)

const (
	dbPath = "./tmp/blocks"
	dbFile = "./tmp/blocks/MANIFEST"
	// GenesisData is the data of the first block
	genesisData = "First Transaction from Genesis" // genesis arbitrary data for input
)

// Blockchain ...
type Blockchain struct {
	LastHash []byte
	Database *badger.DB
}

// Iterator ...
type Iterator struct {
	CurrentHash []byte
	Database    *badger.DB
}

// InitBlockchain init the first Blockchain
func InitBlockchain(address string) *Blockchain {
	if DBexists() {
		fmt.Println("Blockchain already exists")
		runtime.Goexit()
	}

	opts := badger.DefaultOptions(dbPath)
	opts.Dir = dbPath
	opts.ValueDir = dbPath

	db, err := badger.Open(opts)
	Handle(err)

	cbtx := CoinbaseTx(address, genesisData)
	genesis := Genesis(cbtx)
	fmt.Println("Genesis created")

	// set key: genesis.Hash, lh
	err = db.Update(func(txn *badger.Txn) error {
		err = txn.Set(genesis.Hash, genesis.Serialize())
		Handle(err)
		err = txn.Set([]byte("lh"), genesis.Hash)

		return err
	})
	Handle(err)

	lastHash := genesis.Hash

	blockchain := Blockchain{lastHash, db}
	return &blockchain
}

// ContinueBlockchain find lasthash in DB, set lasthash and db into Blockchain, and return it
func ContinueBlockchain(address string) *Blockchain {
	if DBexists() == false {
		fmt.Println("No existing blockchain found, create one!")
		runtime.Goexit()
	}

	opts := badger.DefaultOptions(dbPath)
	opts.Dir = dbPath
	opts.ValueDir = dbPath

	db, err := badger.Open(opts)
	Handle(err)

	var lastHash []byte
	// get lh
	err = db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		Handle(err)

		err = item.Value(func(val []byte) error {
			lastHash = val
			return nil
		})

		return err

	})
	Handle(err)

	chain := Blockchain{lastHash, db}
	return &chain
}

// FindTransaction return the transaction of the given ID
func (chain *Blockchain) FindTransaction(ID []byte) (Transaction, error) {
	iter := chain.CreateIterator()

	for {
		block := iter.Next()

		for _, tx := range block.Transactions {
			if bytes.Compare(tx.ID, ID) == 0 {
				return *tx, nil
			}
		}

		if len(block.PrevHash) == 0 {
			break
		}

	}

	return Transaction{}, errors.New("Transaction does not exist")
}

// SignTransaction sign the transaction with privateKey
func (chain *Blockchain) SignTransaction(tx *Transaction, privKey ecdsa.PrivateKey) {
	prevTXs := make(map[string]Transaction)

	for _, in := range tx.Inputs {
		prevTX, err := chain.FindTransaction(in.ID)
		Handle(err)
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	tx.Sign(privKey, prevTXs)
}

// VerifyTransaction verify the transaction
func (chain *Blockchain) VerifyTransaction(tx *Transaction) bool {
	prevTXs := make(map[string]Transaction)

	for _, in := range tx.Inputs {
		prevTX, err := chain.FindTransaction(in.ID)
		Handle(err)
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	return tx.Verify(prevTXs)
}

// FindUTXO return mapping of address to TxOutputs
func (chain *Blockchain) FindUTXO() map[string]TxOutputs {
	UTXO := make(map[string]TxOutputs)
	spentTXOs := make(map[string][]int)

	iter := chain.CreateIterator()

	for {
		block := iter.Next()

		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)

		Outputs:
			for outIdx, out := range tx.Outputs {
				if spentTXOs[txID] != nil {
					for _, spentOut := range spentTXOs[txID] {
						if spentOut == outIdx {
							continue Outputs
						}
					}
				}

				outs := UTXO[txID]
				outs.Outputs = append(outs.Outputs, out)
				UTXO[txID] = outs
			}

			if tx.IsCoinbase() == false {
				for _, in := range tx.Inputs {

					inTxID := hex.EncodeToString(in.ID)
					spentTXOs[inTxID] = append(spentTXOs[inTxID], in.Out)

				}
			}
		}

		if len(block.PrevHash) == 0 {
			break
		}
	}

	return UTXO

}

// AddBlock add new block in Blockchain's Block
func (chain *Blockchain) AddBlock(transactions []*Transaction) *Block {
	var lastHash []byte

	err := chain.Database.View(func(txn *badger.Txn) error {
		// get lashHash
		item, err := txn.Get([]byte("lh"))
		Handle(err)

		err = item.Value(func(val []byte) error {
			lastHash = val
			return nil
		})
		return err
	})
	Handle(err)

	// create new block
	newBlock := CreateBlock(transactions, lastHash)

	err = chain.Database.Update(func(txn *badger.Txn) error {

		// set newBlock.Hash and lh
		err = txn.Set(newBlock.Hash, newBlock.Serialize())
		Handle(err)

		err = txn.Set([]byte("lh"), newBlock.Hash)

		// set new LastHash to chain
		chain.LastHash = newBlock.Hash

		return err
	})
	Handle(err)

	return newBlock

}

// CreateIterator return type Iterator for iterating blockchain
func (chain *Blockchain) CreateIterator() *Iterator {
	iter := Iterator{chain.LastHash, chain.Database}
	return &iter
}

// Next return current Block from Iterator and set new iter.CurrentHash
func (iter *Iterator) Next() *Block {
	var block *Block
	err := iter.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get(iter.CurrentHash)
		Handle(err)

		err = item.Value(func(val []byte) error {
			encodedBlock := val
			block = Deserialize(encodedBlock)
			return nil
		})

		return err

	})
	Handle(err)

	iter.CurrentHash = block.PrevHash

	return block
}

// DBexists check blockchain exists
func DBexists() bool {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}

	return true
}
