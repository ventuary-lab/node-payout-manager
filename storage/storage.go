package storage

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

/*
	lastTx - last scan transaction
	lastPaymentTx - last rewards payment
	balance_{height}_{address}- rpd user balance by height
	dust_{lastPaymentTx}_{address} - dusts user by height
*/

const (
	LastTxHeightKey  string = "lastTxHeight"
	LastPaymentTxKey string = "lastPaymentTx"
	BalanceKey       string = "balance"
	DustKey          string = "dust"
	DbPath                  = "db/"
)

func allByPrefix(db *leveldb.DB, prefix string) (map[string][]byte, error) {
	result := make(map[string][]byte)

	iter := db.NewIterator(util.BytesPrefix([]byte(prefix)), nil)
	for iter.Next() {
		value := make([]byte, len(iter.Value()))
		copy(value, iter.Value())
		result[string(iter.Key())] = value
	}
	iter.Release()

	if err := iter.Error(); err != nil {
		return nil, err
	}

	return result, nil
}
