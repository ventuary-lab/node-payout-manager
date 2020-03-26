package client

const (
	StakingCalculatorIndexPath string = "/"
)

type StakingCalculator struct {
	Url *string
}

type StakingCalculationPayment struct {
	Recipient *string
	Amount int64
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

func (sc *StakingCalculator) FetchStakingRewards (payments StakingCalculationPayment) *StakingCalculationResult {
	//bytes, err := json.Marshal(payments)
	//
	//if err != nil {
	//	return nil
	//}
	//
	//body := io.Reader
	//
	//response, respErr := http.Post(*sc.Url, "application/json", body)
	return nil
}