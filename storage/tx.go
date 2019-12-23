package storage

import (
	"encoding/binary"

	"github.com/syndtr/goleveldb/leveldb"
)

func LastScanHeight(db *leveldb.DB) (int, error) {
	value, err := db.Get([]byte(LastScanHeightKey), nil)
	if err != nil {
		return 0, err
	}

	return int(binary.LittleEndian.Uint32(value)), err
}
func LastPaymentHeight(db *leveldb.DB) (int, error) {
	value, err := db.Get([]byte(LastPaymentHeightKey), nil)
	if err != nil {
		return 0, err
	}

	return int(binary.LittleEndian.Uint32(value)), err
}

func PutPaymentHeight(db *leveldb.DB, txHash string) error {
	return db.Put([]byte(LastPaymentHeightKey), []byte(txHash), nil)
}
func PutScanHeight(db *leveldb.DB, height int) error {
	b := make([]byte, 8)

	binary.LittleEndian.PutUint32(b, uint32(height))
	return db.Put([]byte(LastScanHeightKey), b, nil)
}
