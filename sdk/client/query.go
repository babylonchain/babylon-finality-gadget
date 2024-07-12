package client

import (
	"strings"

	"github.com/babylonchain/babylon-finality-gadget/sdk/cwclient"
)

func (sdkClient *SdkClient) QueryIsBlockBabylonFinalized(queryParams *cwclient.L2Block) (bool, error) {
	// check if the finality gadget is enabled
	// if not, always return true to pass through op derivation pipeline
	isEnabled, err := sdkClient.cwClient.QueryIsEnabled()
	if err != nil {
		return false, err
	}
	if !isEnabled {
		return true, nil
	}

	// trim prefix 0x for the L2 block hash
	queryParams.BlockHash = strings.TrimPrefix(queryParams.BlockHash, "0x")

	// get the consumer chain id
	consumerId, err := sdkClient.cwClient.QueryConsumerId()
	if err != nil {
		return false, err
	}

	// get all the FPs pubkey for the consumer chain
	allFpPks, err := sdkClient.bbnClient.QueryAllFpBtcPubKeys(consumerId)
	if err != nil {
		return false, err
	}

	// convert the L2 timestamp to BTC height
	btcblockHeight, err := sdkClient.BtcClient.GetBlockHeightByTimestamp(queryParams.BlockTimestamp)
	if err != nil {
		return false, err
	}

	// get all FPs voting power at this BTC height
	allFpPower, err := sdkClient.bbnClient.QueryMultiFpPower(allFpPks, btcblockHeight)
	if err != nil {
		return false, err
	}

	// calculate total voting power
	var totalPower uint64 = 0
	for _, power := range allFpPower {
		totalPower += power
	}

	// no FP has voting power for the consumer chain
	if totalPower == 0 {
		return false, ErrNoFpHasVotingPower
	}

	// get all FPs that voted this (L2 block height, L2 block hash) combination
	votedFpPks, err := sdkClient.cwClient.QueryListOfVotedFinalityProviders(queryParams)
	if err != nil {
		return false, err
	}
	if votedFpPks == nil {
		return false, nil
	}
	// calculate voted voting power
	var votedPower uint64 = 0
	for _, key := range votedFpPks {
		if power, exists := allFpPower[key]; exists {
			votedPower += power
		}
	}

	// quorom < 2/3
	if votedPower*3 < totalPower*2 {
		return false, nil
	}
	return true, nil
}

// TODO: add QueryVP
