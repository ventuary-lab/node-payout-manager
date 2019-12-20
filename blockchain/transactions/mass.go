package transactions

type Transfer struct {
	Recipient string `structs:"recipient"`
	Amount    int64  `structs:"amount"`
}
type MassTransferBody struct {
	AssetID   *string    `structs:"assetId"`
	Transfers []Transfer `structs:"transfers"`
}

func (tx *Transaction) NewMassTransfer(transfers []Transfer, assetId *string) {
	tx.MassTransferBody = &MassTransferBody{
		AssetID:   assetId,
		Transfers: transfers,
	}
	tx.Fee = 50000*len(transfers) + 100000
}
