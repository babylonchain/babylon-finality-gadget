//go:build e2e_op
// +build e2e_op

package e2etest

import (
	"encoding/hex"
	"math/rand"
	"testing"

	"github.com/babylonchain/babylon-da-sdk/sdk"
	"github.com/babylonchain/babylon/testutil/datagen"
	ftypes "github.com/babylonchain/babylon/x/finality/types"
	e2etest "github.com/babylonchain/finality-provider/itest"
	e2etestop "github.com/babylonchain/finality-provider/itest/opstackl2"
	"github.com/babylonchain/finality-provider/types"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/stretchr/testify/require"
)

// tests the query whether the block is Babylon finalized
func TestBlockBabylonFinalized(t *testing.T) {
	stm := StartSdkTestManager(t)
	defer stm.Stop(t)

	// register FP(s) in Babylon Chain and start
	n := 1
	fpList := stm.StartFinalityProvider(t, n)

	var pubRandListInfo *e2etestop.PubRandListInfo
	var msgPub *ftypes.MsgCommitPubRandList
	var mockHash []byte
	// submit BTC delegations for each finality-provider
	for _, fp := range fpList {
		// commit pub rand to smart contract
		pubRandListInfo, msgPub = stm.CommitPubRandList(t, fp.GetBtcPkBIP340())

		// check the public randomness is committed
		stm.WaitForFpPubRandCommitted(t, fp.GetBtcPkBIP340())
		// send a BTC delegation
		_ = stm.InsertBTCDelegation(t, []*btcec.PublicKey{fp.GetBtcPk()}, e2etest.StakingTime, e2etest.StakingAmount)
	}

	// check the BTC delegations are pending
	delsResp := stm.WaitForNPendingDels(t, n)
	require.Equal(t, n, len(delsResp))

	// send covenant sigs to each of the delegations
	for _, delResp := range delsResp {
		d, err := e2etest.ParseRespBTCDelToBTCDel(delResp)
		require.NoError(t, err)
		// send covenant sigs
		stm.InsertCovenantSigForDelegation(t, d)
	}

	// check the BTC delegations are active
	_ = stm.WaitForNActiveDels(t, n)

	for _, fp := range fpList {
		// mock block
		r := rand.New(rand.NewSource(1))
		mockHash := datagen.GenRandomByteArray(r, 32)
		block := &types.BlockInfo{
			Height: uint64(1),
			Hash:   mockHash,
		}
		// fp sign
		fpSig, err := fp.SignFinalitySig(block)
		require.NoError(t, err)

		// pub rand proof
		proof, err := pubRandListInfo.ProofList[0].ToProto().Marshal()
		require.NoError(t, err)

		// submit finality signature to smart contract
		submitRes, err := stm.OpL2ConsumerCtrl.SubmitFinalitySig(
			msgPub.FpBtcPk.MustToBTCPK(),
			block,
			pubRandListInfo.PubRandList[0],
			proof,
			fpSig.ToModNScalar(),
		)
		require.NoError(t, err)
		t.Logf("Submit finality signature to op finality contract %s", submitRes.TxHash)
	}

	queryParams := sdk.QueryParams{
		BlockHeight:    uint64(1),
		BlockHash:      hex.EncodeToString(mockHash),
		BlockTimestamp: uint64(1231473952),
	}
	finalized, err := stm.sdkBBNClient.QueryIsBlockBabylonFinalized(queryParams)
	require.NoError(t, err)
	require.Equal(t, true, finalized)
}
