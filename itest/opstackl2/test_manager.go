//go:build e2e_op
// +build e2e_op

package e2etest

import (
	"testing"

	"github.com/babylonchain/babylon-da-sdk/sdk"
	e2etestbnb "github.com/babylonchain/finality-provider/itest/babylon"
	e2etestop "github.com/babylonchain/finality-provider/itest/opstackl2"
	"github.com/stretchr/testify/require"
)

type OpConsumerTestManager = e2etestop.OpL2ConsumerTestManager
type BBNTestManager = e2etestbnb.TestManager

type SdkTestManager struct {
	OpConsumerTestManager
	BBNTestManager
	sdkBBNClient *sdk.BabylonQueryClient
}

func StartSdkTestManager(t *testing.T) *SdkTestManager {
	ctm := e2etestop.StartOpL2ConsumerManager(t)
	client, err := sdk.NewClient(sdk.Config{
		ChainType:    -1, // only for the e2e test
		ContractAddr: ctm.OpL2ConsumerCtrl.Cfg.OPFinalityGadgetAddress,
	})
	require.NoError(t, err)
	// set test dir
	stm := &SdkTestManager{
		OpConsumerTestManager: *ctm,
		sdkBBNClient:          client,
	}
	return stm
}

func (stm *SdkTestManager) Stop(t *testing.T) {
	stm.OpConsumerTestManager.Stop(t)
}
