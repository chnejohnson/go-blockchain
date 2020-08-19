package blockchain

import (
	"bytes"
	"encoding/hex"
	"log"

	"github.com/dgraph-io/badger"
)

var (
	utxoPrefix   = []byte("utxo-")
	prefixLength = len(utxoPrefix)
)

// UTXOSet for access blockchain database
type UTXOSet struct {
	Blockchain *Blockchain
}

// FindSpendableOutputs take address we want to check and amount we want to send
func (u UTXOSet) FindSpendableOutputs(pubKeyHash []byte, amount int) (int, map[string][]int) {
	unspendOuts := make(map[string][]int)
	accumulated := 0
	db := u.Blockchain.Database

	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(utxoPrefix); it.ValidForPrefix(utxoPrefix); it.Next() {
			item := it.Item()
			k := item.Key()

			err := item.Value(func(val []byte) error {
				k = bytes.TrimPrefix(k, utxoPrefix)
				txID := hex.EncodeToString(k)
				outs := DeserializeOutputs(val)

				for outIdx, out := range outs.Outputs {
					if out.IsLockedWithKey(pubKeyHash) && accumulated < amount {
						accumulated += out.Value
						unspendOuts[txID] = append(unspendOuts[txID], outIdx)

					}
				}
				return nil
			})
			Handle(err)
		}
		return nil
	})
	Handle(err)

	return accumulated, unspendOuts
}

// FindUnspentTransactions find the transactions which have unspent output
func (u UTXOSet) FindUnspentTransactions(pubKeyHash []byte) []TxOutput {
	var UTXOs []TxOutput

	db := u.Blockchain.Database

	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(utxoPrefix); it.ValidForPrefix(utxoPrefix); it.Next() {
			item := it.Item()
			err := item.Value(func(val []byte) error {
				outs := DeserializeOutputs(val)

				for _, out := range outs.Outputs {
					if out.IsLockedWithKey(pubKeyHash) {
						UTXOs = append(UTXOs, out)
					}
				}
				return nil
			})
			Handle(err)

		}
		return nil
	})
	Handle(err)

	return UTXOs
}

// Reindex clear all prefix outputs and create new unspent transaction outputs
func (u UTXOSet) Reindex() {
	db := u.Blockchain.Database

	u.DeleteByPrefix(utxoPrefix)

	UTXO := u.Blockchain.FindUTXO()

	err := db.Update(func(txn *badger.Txn) error {
		for txID, outs := range UTXO {
			key, err := hex.DecodeString(txID)
			if err != nil {
				return err
			}

			key = append(utxoPrefix, key...)

			err = txn.Set(key, outs.Serialize())
			Handle(err)
		}

		return nil
	})

	Handle(err)
}

// Update 主要在更新資料庫的 output，例如幫 output 加上 prefix
func (u *UTXOSet) Update(block *Block) {
	db := u.Blockchain.Database

	err := db.Update(func(txn *badger.Txn) error {
		for _, tx := range block.Transactions {
			if tx.IsCoinbase() == false {
				for _, in := range tx.Inputs {
					updatedOuts := TxOutputs{}
					inID := append(utxoPrefix, in.ID...)
					item, err := txn.Get(inID)
					Handle(err)

					var outs TxOutputs
					err = item.Value(func(val []byte) error {
						outs = DeserializeOutputs(val)
						return nil
					})
					Handle(err)

					for outIdx, out := range outs.Outputs {
						// 此 output index 沒有列在 input.Out 裡頭，代表此 output 為 unspent
						if outIdx != in.Out {
							updatedOuts.Outputs = append(updatedOuts.Outputs, out)
						}
					}

					if len(updatedOuts.Outputs) == 0 {
						// 若沒有任何 unspent output，就刪掉這個 transaction
						if err := txn.Delete(inID); err != nil {
							log.Panic(err)
						}
					} else {
						// 將 unspent output 寫入資料庫
						if err := txn.Set(inID, updatedOuts.Serialize()); err != nil {
							log.Panic(err)
						}
					}
				}
			}

			// 處理 coinbase input 產生的新 outputs (以此獎勵挖礦)
			newOutputs := TxOutputs{}
			for _, out := range tx.Outputs {
				newOutputs.Outputs = append(newOutputs.Outputs, out)
			}

			// 將新的 outputs 存入資料庫
			txID := append(utxoPrefix, tx.ID...)
			if err := txn.Set(txID, newOutputs.Serialize()); err != nil {
				log.Panic(err)
			}
		}

		return nil
	})

	Handle(err)
}

// CountTransactions counts the transactions with unspent outputs
func (u *UTXOSet) CountTransactions() int {
	db := u.Blockchain.Database
	counter := 0

	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions

		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(utxoPrefix); it.ValidForPrefix(utxoPrefix); it.Next() {
			counter++
		}
		return nil
	})
	Handle(err)
	return counter
}

// DeleteByPrefix delete all data with prefix
func (u *UTXOSet) DeleteByPrefix(prefix []byte) {
	// 建立一個可一次刪除多筆資料的 function
	deleteKeys := func(keysForDelete [][]byte) error {
		if err := u.Blockchain.Database.Update(func(txn *badger.Txn) error {
			for _, key := range keysForDelete {
				if err := txn.Delete(key); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
		return nil
	}

	collectSize := 100000
	u.Blockchain.Database.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		keysForDelete := make([][]byte, 0, collectSize)
		keysCollected := 0

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			key := it.Item().KeyCopy(nil)
			keysForDelete = append(keysForDelete, key)
			keysCollected++
			if keysCollected == collectSize {
				if err := deleteKeys(keysForDelete); err != nil {
					log.Panic(err)
				}
				keysForDelete = make([][]byte, 0, collectSize)
				keysCollected = 0
			}
		}

		if keysCollected > 0 {
			if err := deleteKeys(keysForDelete); err != nil {
				log.Panic(err)
			}
		}

		return nil
	})
}
