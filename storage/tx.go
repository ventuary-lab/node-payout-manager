package storage

import (
	"encoding/binary"

	"github.com/syndtr/goleveldb/leveldb"
)

func LastPaymentTx(db *leveldb.DB) (string, error) {
	value, err := db.Get([]byte(LastPaymentTxKey), nil)
	if err != nil {
		return "", err
	}

	return string(value), err
}
func LastTxHeight(db *leveldb.DB) (int, error) {
	value, err := db.Get([]byte(LastTxHeightKey), nil)
	if err != nil {
		return 0, err
	}

	return int(binary.LittleEndian.Uint32(value)), err
}

func PutPaymentTx(db *leveldb.DB, txHash string) error {
	return db.Put([]byte(LastPaymentTxKey), []byte(txHash), nil)
}
func PutLastTxHeight(db *leveldb.DB, height int) error {
	b := make([]byte, 8)

	binary.LittleEndian.PutUint32(b, uint32(height))
	return db.Put([]byte(LastTxHeightKey), b, nil)
}
