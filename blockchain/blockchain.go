package blockchain

import (
	"encoding/hex"
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

// FindUTXO return a slice of unspent Outputs
func (chain *Blockchain) FindUTXO(address string) []TxOutput {
	var UTXOs []TxOutput
	unspentTransactions := chain.FindUnspentTransactions(address)

	for _, utx := range unspentTransactions {
		for _, out := range utx.Outputs {
			if out.CanBeUnlocked(address) {
				UTXOs = append(UTXOs, out)
			}
		}
	}

	return UTXOs
}

// FindSpendableOutputs take address we want to check and amount we want to send
func (chain *Blockchain) FindSpendableOutputs(address string, amount int) (int, map[string][]int) {
	unspendOuts := make(map[string][]int)
	unspentTxs := chain.FindUnspentTransactions(address)
	accumulated := 0

Work:
	for _, utx := range unspentTxs {
		// EncodeToString for turning  txID ([]byte) into map key (string)
		txID := hex.EncodeToString(utx.ID)

		for outIdx, out := range utx.Outputs {
			if out.CanBeUnlocked(address) && accumulated < amount {
				accumulated += out.Value
				unspendOuts[txID] = append(unspendOuts[txID], outIdx)

				// 當 acc 超過輸入的金額就停止遞迴，只輸出夠用的 unspendOuts
				if accumulated >= amount {
					break Work
				}
			}
		}
	}

	return accumulated, unspendOuts
}

// FindUnspentTransactions find the transactions which have unspent output
func (chain *Blockchain) FindUnspentTransactions(address string) []Transaction {
	var unspentTxs []Transaction
	spentTXOs := make(map[string][]int)

	iter := chain.CreateIterator()

	for {
		// Blocks Looping
		block := iter.Next()

		// Transactions Looping
		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)

		Outputs:
			// Outputs Looping
			for outIdx, out := range tx.Outputs {
				if spentTXOs[txID] != nil {
					// In the specific transaction ID, loop all index of spent Outputs
					for _, spentOut := range spentTXOs[txID] {
						// if spentOut equal to the output's index, the output is spent,
						// so that continue the loop because we do not find the unspent transactions
						if spentOut == outIdx {
							continue Outputs
						}
					}
				}

				// 檢查是否有找到 unpent Transaction
				if out.CanBeUnlocked(address) {
					// find the unspent Transaction when the output can be unlocked
					unspentTxs = append(unspentTxs, *tx)
				}
			}

			// Inputs Looping (except for Coinbase)
			if tx.IsCoinbase() == false {
				for _, in := range tx.Inputs {
					// 更新 spent Transaction mapping
					if in.CanUnlock(address) {
						inTxID := hex.EncodeToString(in.ID)
						spentTXOs[inTxID] = append(spentTXOs[inTxID], in.Out)
					}
				}
			}
		}

		// Blocks Looping until meet genesis block
		if len(block.PrevHash) == 0 {
			break
		}
	}

	return unspentTxs
}

// AddBlock add new block in Blockchain's Block
func (chain *Blockchain) AddBlock(transactions []*Transaction) {
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
