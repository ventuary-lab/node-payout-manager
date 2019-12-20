package rpd

import (
	"math"
	"strings"

	"github.com/syndtr/goleveldb/leveldb"

	"github.com/ventuary-lab/node-payout-manager/storage"

	"github.com/ventuary-lab/node-payout-manager/blockchain/transactions"

	"github.com/ventuary-lab/node-payout-manager/state"

	"github.com/ventuary-lab/node-payout-manager/assets"
	"github.com/ventuary-lab/node-payout-manager/blockchain/neutrino"
	"github.com/ventuary-lab/node-payout-manager/client"
)

func SwapAllBalance(node client.Node, sender string, neutrinoContract string) (string, error) {
	var subtrahend float64 = neutrino.InvokeFee + neutrino.MaxTransferFeeSafe
	balance, err := node.GetBalance(sender, assets.WavesAssetId)
	if err != nil {
		return "", err
	}

	if balance < neutrino.MinSwapWavesAmount+subtrahend {
		return "", nil
	}

	tx := neutrino.CreateSwapToNeutrinoTx(sender, neutrinoContract, balance-subtrahend)
	if err := node.SignTx(&tx); err != nil {
		return "", err
	}
	if err := node.Broadcast(tx); err != nil {
		return "", err
	}

	errChan := node.WaitTx(tx.ID)
	if err := <-errChan; err != nil {
		return "", err
	}

	return tx.ID, nil
}

func RecoveryBalance(node client.Node, rpdContract string, assetId string, balances storage.BalanceMap, height int, lastTxHeight int) (map[int]storage.BalanceMap, error) {
	var invokeTxs []transactions.Transaction
	lastTxHash := ""
getTxLoop:
	for {
		txs, err := node.GetTransactions(rpdContract, lastTxHash)
		if err != nil {
			return nil, err
		}

		if txs == nil {
			break getTxLoop
		}
		for _, v := range txs {
			if v.Height < lastTxHeight {
				break getTxLoop
			} else {
				invokeTxs = append(invokeTxs, v)
			}
			lastTxHash = v.ID
		}
	}

	balanceByHeight := make(map[int]storage.BalanceMap)
	groupedTxs := transactions.GroupByHeightAndFunc(invokeTxs)
	for i := height; i > lastTxHeight; i-- {
		balanceByHeight[i] = make(storage.BalanceMap)
		balances.Copy(balanceByHeight[i])
		for _, v := range groupedTxs[i][neutrino.LockRPDFunc] {
			if v.InvokeScriptBody.DApp != rpdContract || len(v.InvokeScriptBody.Payment) != 1 || *v.InvokeScriptBody.Payment[0].AssetId != assetId {
				continue
			}
			balances[v.Sender] -= float64(v.InvokeScriptBody.Payment[0].Amount)
		}
		for _, v := range groupedTxs[i][neutrino.UnlockRPDFunc] {
			if v.InvokeScriptBody.DApp != rpdContract || v.InvokeScriptBody.Call.Args[1].Value.(string) != assetId {
				continue
			}
			balances[v.Sender] += v.InvokeScriptBody.Call.Args[0].Value.(float64)
		}
	}
	return balanceByHeight, nil
}

func CalculateRewords(db *leveldb.DB, totalProfit float64, height int, paymentHeight int) (storage.BalanceMap, error) {
	period := height - paymentHeight
	profitByBlock := totalProfit / float64(period)
	rewords := make(storage.BalanceMap)
	for i := paymentHeight + 1; i <= height; i++ {
		balances, err := storage.Balances(db, i)
		if err != nil {
			return nil, err
		}
		var totalBalance float64
		for _, v := range balances {
			totalBalance += v
		}

		for k, v := range balances {
			share := v / totalBalance
			rewords[k] += share * profitByBlock
		}
	}
	return rewords, nil
}

func StateToBalanceMap(contractState map[string]state.State, neutrinoAssetId string) storage.BalanceMap {
	balances := make(storage.BalanceMap)
	for key, value := range contractState {
		args := strings.Split(key, "_")
		if len(args) != 4 || args[0] != "rpd" || args[1] != "balance" || args[2] != neutrinoAssetId {
			continue
		}
		amount, ok := value.Value.(float64)
		if ok {
			balances[args[3]] = amount
		}
	}
	return balances
}

func CreateMassRewordTx(rewords storage.BalanceMap, sender string, assetId string) transactions.Transaction {
	var transfers []transactions.Transfer
	total := float64(0)
	for address, value := range rewords {
		roundValue := math.Round(value)
		if roundValue > 0 {
			total += roundValue
			transfers = append(transfers, transactions.Transfer{Amount: int64(roundValue), Recipient: address})
		}
	}

	rewordTx := transactions.New(transactions.MassTransfer, sender)
	rewordTx.NewMassTransfer(transfers, &assetId)
	return rewordTx
}
