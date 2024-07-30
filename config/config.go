package config

import (
	"github.com/spf13/viper"
)

const (
	BabylonLocalnet = "chain-test"
	BabylonDevnet   = "euphrates-0.4.0"
)

// Config is the main config for the Babylon finality gadget server
type Config struct {
	Babylon     *BabylonConfig
	Bitcoin     *BitcoinConfig
	DB          *DBConfig
	RpcListener string `mapstructure:"RpcListener" description:"The address to listen on for the gRPC server"`
}

type BabylonConfig struct {
	ContractAddr string `mapstructure:"ContractAddr" description:"Contract address deployed on the Babylon chain"`
	ChainID      string `mapstructure:"ChainID" description:"Chain ID of the Babylon chain"`
	RPCAddr      string `mapstructure:"RPCAddr" description:"RPC address of the Babylon chain"`
}

type BitcoinConfig struct {
	RPCHost string `mapstructure:"RPCHost" description:"The host of the Bitcoin RPC server"`
}

type DBConfig struct {
	DBPath     string `mapstructure:"DBPath" description:"The path to the database file."`
	DBFileName string `mapstructure:"DBFileName" description:"The name of the database file."`
}

func LoadConfig(cfgPath string) (*Config, error) {
	viper.SetConfigFile(cfgPath)
	viper.SetConfigType("toml")

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
