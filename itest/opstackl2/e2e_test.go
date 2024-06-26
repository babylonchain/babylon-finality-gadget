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
	n := 2
	fpList := stm.StartFinalityProvider(t, false, n)

	stakingAmount := e2eutils.StakingAmount
	// submit BTC delegations for each finality-provider
	for i := 0; i < n; i++ {
		if i == 0 {
			stakingAmount = 3 * stakingAmount
		}
		// check the public randomness is committed
		e2eutils.WaitForFpPubRandCommitted(t, fpList[i])
		// send a BTC delegation to consumer and Babylon finality providers
		stm.InsertBTCDelegation(t, []*btcec.PublicKey{bbnFpList[0].GetBtcPk(), fpList[i].GetBtcPk()}, e2eutils.StakingTime, stakingAmount)
	}

	// check the BTC delegations are pending
	delsResp := stm.WaitForNPendingDels(t, n)
	require.Equal(t, n, len(delsResp))

	// send covenant sigs to each of the delegations
	for _, delResp := range delsResp {
		d, err := e2etestbnb.ParseRespBTCDelToBTCDel(delResp)
		require.NoError(t, err)
		// send covenant sigs
		stm.InsertCovenantSigForDelegation(t, d)
	}

	// check the BTC delegations are active
	_ = stm.WaitForNActiveDels(t, n)

	queryFpPk := fpList[n-1].GetBtcPk()
	t.Logf("Query pub rand via finality provider %s", fpList[n-1].GetBtcPkHex())
	var lastCommittedStartHeight uint64
	var mockHash, mockNextHash []byte
	r := rand.New(rand.NewSource(1))
	for i := 0; i < n; i++ {
		t.Logf("Finality provider %s", fpList[i].GetBtcPkHex())
		// query pub rand
		committedPubRandMap, err := stm.OpL2ConsumerCtrl.QueryLastCommittedPublicRand(queryFpPk, 1)
		require.NoError(t, err)
		for key := range committedPubRandMap {
			lastCommittedStartHeight = key
			break
		}
		t.Logf("Last committed pubrandList startHeight %d", lastCommittedStartHeight)

		pubRandList, err := fpList[i].GetPubRandList(lastCommittedStartHeight, stm.FpConfig.NumPubRand)
		require.NoError(t, err)
		// generate commitment and proof for each public randomness
		_, proofList := types.GetPubRandCommitAndProofs(pubRandList)

		// mock block hash
		mockHash = datagen.GenRandomByteArray(r, 32)
		if i == 0 {
			block := &types.BlockInfo{
				Height: lastCommittedStartHeight,
				Hash:   mockHash,
			}
			// fp sign
			fpSig, err := fpList[i].SignFinalitySig(block)
			require.NoError(t, err)

			// pub rand proof
			proof, err := proofList[0].ToProto().Marshal()
			require.NoError(t, err)

			// submit finality signature to smart contract
			_, err = stm.OpL2ConsumerCtrl.SubmitFinalitySig(
				fpList[i].GetBtcPk(),
				block,
				pubRandList[0],
				proof,
				fpSig.ToModNScalar(),
			)
			require.NoError(t, err)
			t.Logf("Submit finality signature to op finality contract")
		}

		// mock next block hash
		mockNextHash = datagen.GenRandomByteArray(r, 32)
		nextBlock := &types.BlockInfo{
			Height: lastCommittedStartHeight + 1,
			Hash:   mockNextHash,
		}
		// fp sign
		fpSig, err := fpList[i].SignFinalitySig(nextBlock)
		require.NoError(t, err)

		// pub rand proof
		proof, err := proofList[1].ToProto().Marshal()
		require.NoError(t, err)

		// submit finality signature to smart contract
		_, err = stm.OpL2ConsumerCtrl.SubmitFinalitySig(
			fpList[i].GetBtcPk(),
			nextBlock,
			pubRandList[1],
			proof,
			fpSig.ToModNScalar(),
		)
		require.NoError(t, err)
		t.Logf("Submit finality signature to op finality contract")

	}

	queryParams := sdk.QueryParams{
		BlockHeight:    lastCommittedStartHeight,
		BlockHash:      hex.EncodeToString(mockHash),
		BlockTimestamp: uint64(1231473952),
	}
	finalized, err := stm.sdkBBNClient.QueryIsBlockBabylonFinalized(queryParams)
	require.NoError(t, err)
	require.Equal(t, true, finalized)

	queryParams = sdk.QueryParams{
		BlockHeight:    lastCommittedStartHeight + 1,
		BlockHash:      hex.EncodeToString(mockNextHash),
		BlockTimestamp: uint64(1231473952),
	}
	finalized, err = stm.sdkBBNClient.QueryIsBlockBabylonFinalized(queryParams)
	require.NoError(t, err)
	require.Equal(t, false, finalized)
}
