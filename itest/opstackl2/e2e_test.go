//go:build e2e_op
// +build e2e_op

package e2etest

import (
	"encoding/hex"
	"math/rand"
	"testing"

	"github.com/babylonchain/babylon-da-sdk/sdk"
	"github.com/babylonchain/babylon/testutil/datagen"
	e2eutils "github.com/babylonchain/finality-provider/itest"
	e2etestbnb "github.com/babylonchain/finality-provider/itest/babylon"
	"github.com/babylonchain/finality-provider/types"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/stretchr/testify/require"
)

// tests the query whether the block is Babylon finalized
func TestBlockBabylonFinalized(t *testing.T) {
	stm := StartSdkTestManager(t)
	defer stm.Stop(t)

	// A BTC delegation has to stake to at least one Babylon finality provider
	// https://github.com/babylonchain/babylon-private/blob/base/consumer-chain-support/x/btcstaking/keeper/msg_server.go#L169-L213
	// So we have to start Babylon chain FP
	bbnFpList := stm.StartFinalityProvider(t, true, 1)

	// start consumer chain FP
	n := 1
	fpList := stm.StartFinalityProvider(t, false, n)

	var lastCommittedStartHeight uint64
	var mockHash []byte
	// submit BTC delegations for each finality-provider
	for _, fp := range fpList {
		// check the public randomness is committed
		e2eutils.WaitForFpPubRandCommitted(t, fp)
		// send a BTC delegation too consumer finality provider
		// send a BTC delegation to Babylon finality provider
		stm.InsertBTCDelegation(t, []*btcec.PublicKey{bbnFpList[0].GetBtcPk(), fp.GetBtcPk()}, e2eutils.StakingTime, e2eutils.StakingAmount)
	}

	// check the BTC delegations are pending
	delsResp := stm.WaitForNPendingDels(t, 1)
	require.Equal(t, 1, len(delsResp))

	// send covenant sigs to each of the delegations
	for _, delResp := range delsResp {
		d, err := e2etestbnb.ParseRespBTCDelToBTCDel(delResp)
		require.NoError(t, err)
		// send covenant sigs
		stm.InsertCovenantSigForDelegation(t, d)
	}

	// check the BTC delegations are active
	_ = stm.WaitForNActiveDels(t, 1)

	for _, fp := range fpList {
		// query pub rand
		committedPubRandMap, err := stm.OpL2ConsumerCtrl.QueryLastCommittedPublicRand(fp.GetBtcPk(), 1)
		require.NoError(t, err)
		for key := range committedPubRandMap {
			lastCommittedStartHeight = key
			break
		}
		t.Logf("Last committed pubrandList startHeight %d", lastCommittedStartHeight)

		pubRandList, err := fp.GetPubRandList(lastCommittedStartHeight, stm.OpConsumerTestManager.FpConfig.NumPubRand)
		require.NoError(t, err)
		// generate commitment and proof for each public randomness
		_, proofList := types.GetPubRandCommitAndProofs(pubRandList)

		// mock block hash
		r := rand.New(rand.NewSource(1))
		mockHash := datagen.GenRandomByteArray(r, 32)
		block := &types.BlockInfo{
			Height: lastCommittedStartHeight,
			Hash:   mockHash,
		}
		// fp sign
		fpSig, err := fp.SignFinalitySig(block)
		require.NoError(t, err)

		// pub rand proof
		proof, err := proofList[0].ToProto().Marshal()
		require.NoError(t, err)

		// submit finality signature to smart contract
		submitRes, err := stm.OpL2ConsumerCtrl.SubmitFinalitySig(
			fp.GetBtcPk(),
			block,
			pubRandList[0],
			proof,
			fpSig.ToModNScalar(),
		)
		require.NoError(t, err)
		t.Logf("Submit finality signature to op finality contract %s", submitRes.TxHash)
	}

	queryParams := sdk.QueryParams{
		BlockHeight:    lastCommittedStartHeight,
		BlockHash:      hex.EncodeToString(mockHash),
		BlockTimestamp: uint64(1231473952),
	}
	finalized, err := stm.sdkBBNClient.QueryIsBlockBabylonFinalized(queryParams)
	require.NoError(t, err)
	require.Equal(t, true, finalized)
}
