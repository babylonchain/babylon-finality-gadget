package config

import (
	"fmt"

	"github.com/babylonchain/babylon-finality-gadget/btcclient"
)

const (
	BabylonLocalnet = -1
	BabylonTestnet  = 0
	BabylonMainnet  = 1
)

// Config defines configuration for the Babylon query client
type Config struct {
	BTCConfig    *btcclient.BTCConfig `mapstructure:"btc-config"`
	ContractAddr string               `mapstructure:"contract-addr"`
	ChainType    int                  `mapstructure:"chain-type"`
}

func (config *Config) GetRpcAddr() (string, error) {
	switch config.ChainType {
	case BabylonLocalnet:
		// only for the e2e test
		return "http://127.0.0.1:26657", nil
	case BabylonTestnet:
		return "https://rpc-euphrates.devnet.babylonchain.io/", nil
	// TODO: replace with babylon RPCs when QuerySmartContractStateRequest query is supported
	case BabylonMainnet:
		return "https://rpc-euphrates.devnet.babylonchain.io/", nil
	default:
		return "", fmt.Errorf("unrecognized chain type: %d", config.ChainType)
	}
}
