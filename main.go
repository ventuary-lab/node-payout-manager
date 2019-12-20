package main

import (
	"flag"
	"strconv"
	"time"

	"github.com/cryptopay-dev/yaga/logger/zap"

	"github.com/cryptopay-dev/yaga/logger"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/ventuary-lab/node-payout-manager/client"
	"github.com/ventuary-lab/node-payout-manager/config"
	"github.com/ventuary-lab/node-payout-manager/rpd"
	"github.com/ventuary-lab/node-payout-manager/storage"
)

const (
	defaultConfigFileName = "config.json"
)

var currLogger logger.Logger

func main() {
	var platform, confFileName string
	flag.StringVar(&platform, "platform", zap.Production, "set platform config (development/ ")
	flag.StringVar(&confFileName, "config", defaultConfigFileName, "set config path")
	flag.Parse()

	currLogger = zap.New(platform)
	cfg, err := config.Load(confFileName)
	if err != nil {
		currLogger.Error(err)
	}

	var nodeClient = client.New(cfg.NodeURL, cfg.ApiKey)
	for {
		err := Scan(nodeClient, cfg)
		if err != nil {
			currLogger.Error(err)
		}
		time.Sleep(time.Duration(cfg.SleepSec) * time.Second)
	}
}

func Scan(nodeClient client.Node, cfg config.Config) error {
	currLogger.Infow("Start scan")

	// convert all balance waves -> usd-n

	swapHash, err := rpd.SwapAllBalance(nodeClient, cfg.Sender, cfg.NeutrinoContract)
	currLogger.Infow("Await swap status")
	if err != nil {
		return err
	}

	currLogger.Infow("Swap tx: " + swapHash)

	db, err := leveldb.OpenFile(storage.DbPath, nil)
	if err != nil {
		return err
	}
	defer db.Close()

	lastPaymentTxHash, err := storage.LastPaymentTx(db)
	if err != nil && err != leveldb.ErrNotFound {
		return err
	} else if lastPaymentTxHash == "" {
		lastPaymentTxHash = cfg.DefaultLastPaymentTx
	}
	lastPaymentTx, err := nodeClient.GetTxById(lastPaymentTxHash)
	if err != nil {
		return err
	}

	lastTxHeight, err := storage.LastTxHeight(db)
	if err != nil && err != leveldb.ErrNotFound {
		return err
	} else if lastTxHeight == 0 {
		lastTxHeight = lastPaymentTx.Height
	}

	currLogger.Infow("Last payment tx: " + lastPaymentTxHash)
	currLogger.Infow("Last scanned tx height: " + strconv.Itoa(lastTxHeight))

	height, err := nodeClient.GetHeight()
	if err != nil {
		return err
	}

	currLogger.Infow("Height: " + strconv.Itoa(height))

	currLogger.Infow("Get state by address")
	contractState, err := nodeClient.GetStateByAddress(cfg.RPDContract)
	if err != nil {
		return err
	}
	balances := rpd.StateToBalanceMap(contractState, cfg.AssetId)

	currLogger.Infow("Recovery balance")
	balancesByHeight, err := rpd.RecoveryBalance(nodeClient, cfg.RPDContract, cfg.AssetId, balances, height, lastTxHeight)
	if err != nil {
		return err
	}

	currLogger.Infow("Write to level db")
	// write to level db
	for height, balances := range balancesByHeight {
		err := storage.PutBalances(db, height, balances)
		if err != nil {
			return err
		}
	}
	err = storage.PutLastTxHeight(db, height)
	if err != nil {
		return err
	}

	// payout rewords
	if height >= lastPaymentTx.Height+cfg.PayoutInterval {
		balance, err := nodeClient.GetBalance(cfg.Sender, cfg.AssetId)
		if err != nil {
			return err
		}
		currLogger.Infow("Total balance: " + strconv.FormatFloat(balance, 'f', 0, 64))
		currLogger.Infow("Calculate rewords")
		rewords, err := rpd.CalculateRewords(db, balance, height, lastPaymentTx.Height)
		if err != nil {
			return err
		}

		rewordTx := rpd.CreateMassRewordTx(rewords, cfg.Sender)

		currLogger.Infow("Sign and broadcast")
		if err := nodeClient.SignTx(&rewordTx); err != nil {
			return err
		}

		if err := nodeClient.Broadcast(rewordTx); err != nil {
			return err
		}
		//TODO massTx
		if err := storage.PutPaymentTx(db, rewordTx.ID); err != nil {
			return err
		}

		currLogger.Infow("Reword tx: " + rewordTx.ID)
		currLogger.Debugw("Reword tx: %s", rewordTx)
	}
	return nil
}
