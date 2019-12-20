package neutrino

import "github.com/ventuary-lab/node-payout-manager/blockchain/transactions"

const (
	SwapWavesToNeutrinoFunc transactions.ContractFunc = "swapWavesToNeutrino"
	UnlockRPDFunc           transactions.ContractFunc = "unlockNeutrino"
	LockRPDFunc             transactions.ContractFunc = "lockNeutrino"

	MinSwapWavesAmount = 100000000
	InvokeFee          = 500000
	MaxTransferFeeSafe = 100000000
)

func CreateSwapToNeutrinoTx(sender string, neutrinoContract string, wavesAmount float64) transactions.Transaction {
	var tx = transactions.New(transactions.InvokeScript, sender)
	tx.NewInvokeScript(neutrinoContract, transactions.FuncCall{
		Function: SwapWavesToNeutrinoFunc,
		Args:     nil,
	}, []transactions.Payment{
		{
			Amount:  int64(wavesAmount),
			AssetId: nil,
		},
	}, InvokeFee)
	return tx
}
