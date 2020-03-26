package client

import (
	bytes2 "bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
)

const (
	StakingCalculatorIndexPath string = "/"
)

type StakingCalculator struct {
	Url *string
}

type StakingCalculationPayment struct {
	Recipient string 	`json:"recipient"`
	Amount int64        `json:"amount"`
}

type StakingCalculationResult struct {
	Direct []StakingCalculationPayment
	Ref []StakingCalculationPayment
}

func (sc *StakingCalculator) Create (url string) *StakingCalculator {
	return &StakingCalculator{
		Url: &url,
	}
}

func (sc *StakingCalculator) FetchStakingRewards (payments []StakingCalculationPayment) *StakingCalculationResult {
	reqDict := map[string][]StakingCalculationPayment {
		"payments": payments,
	}
	bytes, err := json.Marshal(reqDict)

	if err != nil {
		return nil
	}

	body := bytes2.NewReader(bytes)
	response, _ := http.Post(*sc.Url, "application/json", body)

	var parsed StakingCalculationResult
	responseByteValue, _ := ioutil.ReadAll(response.Body)
	_ = json.Unmarshal(responseByteValue, &parsed)
	// fmt.Printf("CODE: %v %v \n", response.Status, string(bytes))

	defer response.Body.Close()

	return &parsed
}