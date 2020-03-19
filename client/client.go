package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/ventuary-lab/node-payout-manager/assets"

	"github.com/ventuary-lab/node-payout-manager/state"

	"github.com/ventuary-lab/node-payout-manager/blockchain/transactions"
)

const (
	SignPath               = "/transactions/sign"
	BroadcastPath          = "/transactions/broadcast"
	GetTxPath              = "/transactions/info"
	GetUnconfirmedTxByPath = "/transactions/unconfirmed/info"
	GetStateByAddressPath  = "/addresses/data"
	GetHeightPath          = "/blocks/height"
	GetBalancePath         = "/addresses/balance"
	GetBalanceByAssetPath  = "/assets/balance"
	GetTransactionsPath    = "/transactions/address"

	WaitCount      = 10
	DefaultTxLimit = 1000
)

type Node struct {
	nodeUrl string
	apiKey  string
}

func New(nodeUrl string, apiKey string) Node {
	return Node{nodeUrl: nodeUrl, apiKey: apiKey}
}

func (node Node) GetBalance(address string, assetId string) (float64, error) {
	var rsBody []byte
	var err error
	if assetId == assets.WavesAssetId {
		rsBody, _, err = sendRequest("GET", node.nodeUrl+GetBalancePath+"/"+address, nil, "")
	} else {
		rsBody, _, err = sendRequest("GET", node.nodeUrl+GetBalanceByAssetPath+"/"+address+"/"+assetId, nil, "")
	}
	if err != nil {
		return 0, err
	}
	result := make(map[string]interface{})
	if err := json.Unmarshal(rsBody, &result); err != nil {
		return 0, err
	}
	return result["balance"].(float64), nil
}

func (node Node) GetHeight() (int, error) {
	rsBody, _, err := sendRequest("GET", node.nodeUrl+GetHeightPath, nil, "")
	if err != nil {
		return 0, err
	}
	result := make(map[string]interface{})
	if err := json.Unmarshal(rsBody, &result); err != nil {
		return 0, err
	}
	return int(result["height"].(float64)), nil
}

func (node Node) GetStateByAddress(address string) (map[string]state.State, error) {
	rsBody, _, err := sendRequest("GET", node.nodeUrl+GetStateByAddressPath+"/"+address, nil, "")
	if err != nil {
		return nil, err
	}
	states := state.States{}
	if err := json.Unmarshal(rsBody, &states); err != nil {
		return nil, err
	}
	return states.Map(), nil
}
func (node Node) GetTransactions(address string, after string) ([]transactions.Transaction, error) {
	url := node.nodeUrl + GetTransactionsPath + "/" + address + "/limit/" + strconv.Itoa(DefaultTxLimit) + "?after=" + after
	rsBody, _, err := sendRequest("GET", url, nil, "")
	if err != nil {
		return nil, err
	}

	var txsMap [][]map[string]interface{}
	if err := json.Unmarshal(rsBody, &txsMap); err != nil {
		return nil, err
	}

	var txs []transactions.Transaction
	for _, txMap := range txsMap[0] {
		tx, err := transactions.Parse(txMap)
		if err != nil {
			continue
		}
		txs = append(txs, tx)
	}
	return txs, nil
}

func (node Node) GetTxById(id string) (transactions.Transaction, error) {
	rsBody, _, err := sendRequest("GET", node.nodeUrl+GetTxPath+"/"+id, nil, "")
	if err != nil {
		return transactions.Transaction{}, err
	}

	return transactions.Unmarshal(rsBody)
}

func (node Node) IsUnconfirmedTx(id string) (bool, error) {
	_, code, err := sendRequest("GET", node.nodeUrl+GetUnconfirmedTxByPath+"/"+id, nil, "")
	if err != nil && code != 404 {
		return true, err
	}

	return code == 200, nil
}

func (node Node) SignTx(tx *transactions.Transaction) error {
	rqBody, err := tx.Marshal()
	if err != nil {
		return err
	}
	rsBody, _, err := sendRequest("POST", node.nodeUrl+SignPath, rqBody, node.apiKey)
	if err != nil {
		return err
	}

	newTx, err := transactions.Unmarshal(rsBody)
	if err != nil {
		return err
	}
	*tx = newTx
	return nil
}

func (node Node) WaitTx(id string) <-chan error {
	out := make(chan error)
	go func() {
		defer close(out)
		for i := 0; i < WaitCount; i++ {
			un, err := node.IsUnconfirmedTx(id)
			if err != nil {
				out <- err
				break
			}

			if un == false {
				tx, err := node.GetTxById(id)
				if err != nil {
					out <- err
				}
				if tx.ID == "" {
					out <- errors.New("transaction not found")
				} else {
					out <- nil
				}
				break
			}

			if i == (WaitCount - 1) {
				out <- errors.New("transaction not found")
				break
			}

			time.Sleep(time.Second)
		}
	}()
	return out
}

func (node Node) Broadcast(tx transactions.Transaction) error {
	rqBody, err := tx.Marshal()
	if err != nil {
		return err
	}
	_, _, err = sendRequest("POST", node.nodeUrl+BroadcastPath, rqBody, node.apiKey)
	if err != nil {
		return err
	}

	return nil
}

func sendRequest(method string, url string, rqBody []byte, apiKey string) ([]byte, int, error) {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(rqBody))
	req.Header.Add("content-type", "application/json")
	if apiKey != "" {
		req.Header.Add("X-API-Key", apiKey)
	}

	if err != nil {
		return nil, 0, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		if resp != nil {
			return nil, resp.StatusCode, err
		} else {
			return nil, 520, err
		}
	}

	defer resp.Body.Close()
	rsBody, err := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return rsBody, resp.StatusCode, errors.New(string(rsBody))
	}
	return rsBody, resp.StatusCode, nil
}
