package config

import (
	"encoding/json"
	"io/ioutil"
)

type Config struct {
	NodeURL                  string
	ApiKey                   string
	NeutrinoContract         string
	RPDContract              string
	PayoutInterval           int
	Sender                   string
	SleepSec                 int
	AssetId                  string
	DefaultLastPaymentHeight int
	StakingCalculatorUrl     string
}

func Load(filename string) (Config, error) {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return Config{}, err
	}
	config := Config{}
	if err := json.Unmarshal(file, &config); err != nil {
		return Config{}, err
	}
	return config, err
}
