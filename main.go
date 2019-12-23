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
	rpdConfig := rpd.Config{
		Sender:           cfg.Sender,
		NeutrinoContract: cfg.NeutrinoContract,
		AssetId:          cfg.AssetId,
		RpdContract:      cfg.RPDContract,
	}
	currLogger.Infow("Start scan")

	// convert all balance waves -> usd-n

	swapHash, err := rpd.SwapAllBalance(nodeClient, rpdConfig)

	if err != nil {
		return err
	}
	if swapHash != "" {
		errChan := nodeClient.WaitTx(swapHash)
		if err := <-errChan; err != nil {
			return err
		}
		currLogger.Infow("Swap tx: " + swapHash)
	}

	db, err := leveldb.OpenFile(storage.DbPath, nil)
	if err != nil {
		return err
	}
	defer db.Close()

	lastPaymentHeight, err := storage.LastPaymentHeight(db)
	if err != nil && err != leveldb.ErrNotFound {
		return err
	} else if lastPaymentHeight == 0 {
		lastPaymentHeight = cfg.DefaultLastPaymentHeight
	}
	currLogger.Infow("Last payment height: " + strconv.Itoa(lastPaymentHeight))

	lastHeight, err := storage.LastScanHeight(db)
	if err != nil && err != leveldb.ErrNotFound {
		return err
	} else if lastHeight == 0 {
		lastHeight = lastPaymentHeight
	}
	currLogger.Infow("Last scan height: " + strconv.Itoa(lastHeight))

	height, err := nodeClient.GetHeight()
	if err != nil {
		return err
	}

	currLogger.Infow("Height: " + strconv.Itoa(height))

	currLogger.Infow("Get contract state")
	contractState, err := nodeClient.GetStateByAddress(cfg.RPDContract)
	if err != nil {
		return err
	}
	balances := rpd.StateToBalanceMap(contractState, rpdConfig)
	if len(balances) == 0 {
		currLogger.Infow("Neutrino stakers not found")
		return nil
	}
	currLogger.Debug("Contract state: ", balances)

	currLogger.Infow("Recovery balance")
	balancesByHeight, err := rpd.RecoveryBalance(nodeClient, rpdConfig, balances, height, lastHeight)
	if err != nil {
		return err
	}
	currLogger.Debug("Balance: ", balancesByHeight)

	currLogger.Infow("Write to level db")
	for height, balances := range balancesByHeight {
		err := storage.PutBalances(db, height, balances)
		if err != nil {
			return err
		}
	}
	err = storage.PutScanHeight(db, height)
	if err != nil {
		return err
	}

	if height >= lastPaymentHeight+cfg.PayoutInterval {
		currLogger.Infow("Start payout rewords")
		balance, err := nodeClient.GetBalance(cfg.Sender, cfg.AssetId)
		if balance == 0 {
			currLogger.Infow("Await pacemaker oracle or swap")
			return nil
		}
		if err != nil {
			return err
		}
		currLogger.Infow("Total balance: " + strconv.FormatFloat(balance, 'f', 0, 64))
		currLogger.Infow("Calculate rewords")
		rewords, err := rpd.CalculateRewords(db, balance, height, lastPaymentHeight)
		if err != nil {
			return err
		}
		currLogger.Debug("Rewords: ", rewords)

		rewordTx := rpd.CreateMassRewordTx(rewords, rpdConfig)
		currLogger.Infow("Sign and broadcast")
		if err := nodeClient.SignTx(&rewordTx); err != nil {
			return err
		}
		currLogger.Infow("Reword tx hash: " + rewordTx.ID)
		currLogger.Debug("Reword tx: ", rewordTx)

		if err := nodeClient.Broadcast(rewordTx); err != nil {
			return err
		}

		//TODO
		if err := storage.PutPaymentHeight(db, rewordTx.ID); err != nil {
			return err
		}
	}
	currLogger.Infow("End scan")
	return nil
}
