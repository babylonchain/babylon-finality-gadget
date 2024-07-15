package client

import (
	"testing"

	"github.com/babylonchain/babylon-finality-gadget/sdk/cwclient"
	"github.com/babylonchain/babylon-finality-gadget/testutil/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestTestQueryIsBlockBabylonFinalizedFinalityGadgetDisabled(t *testing.T) {
	ctl := gomock.NewController(t)

	// mock CwClient
	mockCwClient := mocks.NewMockCosmWasmClientInterface(ctl)
	mockCwClient.EXPECT().QueryIsEnabled().Return(false, nil).AnyTimes()

	mockSdkClient := &SdkClient{
		cwClient:  mockCwClient,
		bbnClient: nil,
		btcClient: nil,
	}

	// check QueryIsBlockBabylonFinalized always returns true when finality gadget is not enabled
	res, err := mockSdkClient.QueryIsBlockBabylonFinalized(nil)
	require.NoError(t, err)
	require.True(t, res)
}

func TestQueryIsBlockBabylonFinalized(t *testing.T) {
	ctl := gomock.NewController(t)

	queryParams := &cwclient.L2Block{
		BlockHash:      "0x123",
		BlockHeight:    123,
		BlockTimestamp: 12345,
	}

	// mock CwClient
	mockCwClient := mocks.NewMockCosmWasmClientInterface(ctl)
	mockCwClient.EXPECT().QueryIsEnabled().Return(true, nil).AnyTimes()
	const consumerChainID = "consumer-chain-id"
	mockCwClient.EXPECT().QueryConsumerId().Return(consumerChainID, nil).AnyTimes()
	mockCwClient.EXPECT().QueryListOfVotedFinalityProviders(queryParams).Return([]string{"pk1", "pk2"}, nil).AnyTimes()

	// mock BTCClient
	mockBTCClient := mocks.NewMockBTCClientInterface(ctl)
	const BTCHeight = uint64(111)
	mockBTCClient.EXPECT().GetBlockHeightByTimestamp(queryParams.BlockTimestamp).Return(BTCHeight, nil).AnyTimes()

	// mock BBNClient
	mockBBNClient := mocks.NewMockBBNClientInterface(ctl)
	mockBBNClient.EXPECT().QueryAllFpBtcPubKeys(consumerChainID).Return([]string{"pk1", "pk2"}, nil).AnyTimes()
	mockBBNClient.EXPECT().QueryMultiFpPower([]string{"pk1", "pk2"}, BTCHeight).Return(map[string]uint64{"pk1": 50, "pk2": 150}, nil).AnyTimes()

	mockSdkClient := &SdkClient{
		cwClient:  mockCwClient,
		bbnClient: mockBBNClient,
		btcClient: mockBTCClient,
	}

	res, err := mockSdkClient.QueryIsBlockBabylonFinalized(queryParams)
	require.NoError(t, err)
	require.True(t, res)
}
