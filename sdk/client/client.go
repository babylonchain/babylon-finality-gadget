package client

import (
	"fmt"

	"github.com/babylonchain/babylon-finality-gadget/testutils"
	bbncfg "github.com/babylonchain/babylon/client/config"
	"go.uber.org/zap"

	sdkconfig "github.com/babylonchain/babylon-finality-gadget/config"
	"github.com/babylonchain/babylon-finality-gadget/sdk/bbnclient"
	"github.com/babylonchain/babylon-finality-gadget/sdk/btcclient"

	babylonClient "github.com/babylonchain/babylon/client/client"

	"github.com/babylonchain/babylon-finality-gadget/sdk/cwclient"
)

// SdkClient is a client that can only perform queries to a Babylon node
// It only requires the client config to have `rpcAddr`, but not other fields
// such as keyring, chain ID, etc..
type SdkClient struct {
	bbnClient IBabylonClient
	cwClient  ICosmWasmClient
	btcClient IBitcoinClient
}

// NewClient creates a new BabylonFinalityGadgetClient according to the given config
func NewClient(config *sdkconfig.Config) (*SdkClient, error) {
	bbnConfig := bbncfg.DefaultBabylonConfig()
	bbnConfig.RPCAddr = config.Babylon.RPCAddr

	logger, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}

	// Note: We can just ignore the below info which is printed by bbnclient.New
	// service injective.evm.v1beta1.Msg does not have cosmos.msg.v1.service proto annotation
	babylonClient, err := babylonClient.New(
		&bbnConfig,
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Babylon client: %w", err)
	}

	var btcClient IBitcoinClient
	btcConfig := btcclient.DefaultBTCConfig()
	btcConfig.RPCHost = config.Bitcoin.RPCHost
	// Create BTC client
	switch config.Babylon.ChainID {
	// TODO: once we set up our own local BTC devnet, we don't need to use this mock BTC client
	case sdkconfig.BabylonLocalnet:
		btcClient, err = testutils.NewMockBTCClient(btcConfig, logger)
	default:
		btcClient, err = btcclient.NewBTCClient(btcConfig, logger)
	}
	if err != nil {
		return nil, err
	}

	cwClient := cwclient.NewClient(babylonClient.QueryClient.RPCClient, config.Babylon.ContractAddr)

	return &SdkClient{
		bbnClient: &bbnclient.Client{QueryClient: babylonClient.QueryClient},
		cwClient:  cwClient,
		btcClient: btcClient,
	}, nil
}
