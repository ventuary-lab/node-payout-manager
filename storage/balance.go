package storage

import (
	"encoding/binary"
	"math"
	"strconv"
	"strings"

	"github.com/syndtr/goleveldb/leveldb"
)

type BalanceMap map[string]float64

func (amounts BalanceMap) Copy(dst BalanceMap) {
	for k, v := range amounts {
		dst[k] = v
	}
}

func Balances(db *leveldb.DB, height int) (BalanceMap, error) {
	return addressesAmount(db, BalanceKey+"_"+strconv.Itoa(height), balanceAddressByKey)
}
func Dusts(db *leveldb.DB, paymentTx string) (BalanceMap, error) {
	return addressesAmount(db, DustKey+"_"+paymentTx, dustAddressByKey)
}

func PutBalances(db *leveldb.DB, height int, balances BalanceMap) error {
	return putAmounts(db, BalanceKey+"_"+strconv.Itoa(height), balances)
}
func PutDusts(db *leveldb.DB, paymentTx string, balances BalanceMap) error {
	return putAmounts(db, BalanceKey+"_"+paymentTx, balances)
}

func addressesAmount(db *leveldb.DB, key string, addressByKeyFunc func(key string) string) (BalanceMap, error) {
	value, err := allByPrefix(db, key)
	if err != nil {
		return nil, err
	}
	amounts := make(BalanceMap)
	for k, v := range value {
		key := addressByKeyFunc(k)
		bits := binary.LittleEndian.Uint64(v)
		amounts[key] = math.Float64frombits(bits)
	}
	return amounts, nil
}

func putAmounts(db *leveldb.DB, prefix string, balances BalanceMap) error {
	batch := new(leveldb.Batch)
	for k, v := range balances {
		b := make([]byte, 8)

		binary.LittleEndian.PutUint64(b, math.Float64bits(v))
		batch.Put([]byte(prefix+"_"+k), b)
	}

	return db.Write(batch, nil)
}

func balanceAddressByKey(key string) string {
	return strings.Split(key, "_")[2]
}
func dustAddressByKey(key string) string {
	return strings.Split(key, "_")[2]
}
