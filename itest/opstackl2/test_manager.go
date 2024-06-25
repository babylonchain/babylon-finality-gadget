//go:build e2e_op
// +build e2e_op

package e2etest

import (
	"testing"
	"time"

	"math/rand"

	"github.com/babylonchain/babylon-da-sdk/sdk"
	"github.com/babylonchain/babylon/btcstaking"
	asig "github.com/babylonchain/babylon/crypto/schnorr-adaptor-signature"
	"github.com/babylonchain/babylon/testutil/datagen"
	bbntypes "github.com/babylonchain/babylon/types"
	btcctypes "github.com/babylonchain/babylon/x/btccheckpoint/types"
	btclctypes "github.com/babylonchain/babylon/x/btclightclient/types"
	bstypes "github.com/babylonchain/babylon/x/btcstaking/types"
	e2etest "github.com/babylonchain/finality-provider/itest"
	e2etestbnb "github.com/babylonchain/finality-provider/itest/babylon"
	e2etestop "github.com/babylonchain/finality-provider/itest/opstackl2"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/wire"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/stretchr/testify/require"
)

type OpConsumerTestManager = e2etestop.OpL2ConsumerTestManager

type SdkTestManager struct {
	OpConsumerTestManager
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

func (stm *SdkTestManager) InsertBTCDelegation(t *testing.T, fpPks []*btcec.PublicKey, stakingTime uint16, stakingAmount int64) *e2etestbnb.TestDelegationData {
	params := stm.StakingParams
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// delegator BTC key pairs, staking tx and slashing tx
	delBtcPrivKey, delBtcPubKey, err := datagen.GenRandomBTCKeyPair(r)
	require.NoError(t, err)

	unbondingTime := uint16(stm.StakingParams.MinimumUnbondingTime()) + 1
	testStakingInfo := datagen.GenBTCStakingSlashingInfo(
		r,
		t,
		e2etest.BtcNetworkParams,
		delBtcPrivKey,
		fpPks,
		params.CovenantPks,
		params.CovenantQuorum,
		stakingTime,
		stakingAmount,
		params.SlashingAddress.String(),
		params.SlashingRate,
		unbondingTime,
	)

	// delegator Babylon key pairs
	delBabylonPrivKey, delBabylonPubKey, err := datagen.GenRandomSecp256k1KeyPair(r)
	require.NoError(t, err)

	// proof-of-possession
	pop, err := bstypes.NewPoP(delBabylonPrivKey, delBtcPrivKey)
	require.NoError(t, err)

	// create and insert BTC headers which include the staking tx to get staking tx info
	btcTipHeaderResp, err := stm.BBNClient.QueryBtcLightClientTip()
	require.NoError(t, err)
	tipHeader, err := bbntypes.NewBTCHeaderBytesFromHex(btcTipHeaderResp.HeaderHex)
	require.NoError(t, err)
	blockWithStakingTx := datagen.CreateBlockWithTransaction(r, tipHeader.ToBlockHeader(), testStakingInfo.StakingTx)
	accumulatedWork := btclctypes.CalcWork(&blockWithStakingTx.HeaderBytes)
	accumulatedWork = btclctypes.CumulativeWork(accumulatedWork, btcTipHeaderResp.Work)
	parentBlockHeaderInfo := &btclctypes.BTCHeaderInfo{
		Header: &blockWithStakingTx.HeaderBytes,
		Hash:   blockWithStakingTx.HeaderBytes.Hash(),
		Height: btcTipHeaderResp.Height + 1,
		Work:   &accumulatedWork,
	}
	headers := make([]bbntypes.BTCHeaderBytes, 0)
	headers = append(headers, blockWithStakingTx.HeaderBytes)
	for i := 0; i < int(params.ComfirmationTimeBlocks); i++ {
		headerInfo := datagen.GenRandomValidBTCHeaderInfoWithParent(r, *parentBlockHeaderInfo)
		headers = append(headers, *headerInfo.Header)
		parentBlockHeaderInfo = headerInfo
	}
	_, err = stm.BBNClient.InsertBtcBlockHeaders(headers)
	require.NoError(t, err)
	btcHeader := blockWithStakingTx.HeaderBytes
	serializedStakingTx, err := bbntypes.SerializeBTCTx(testStakingInfo.StakingTx)
	require.NoError(t, err)
	txInfo := btcctypes.NewTransactionInfo(&btcctypes.TransactionKey{Index: 1, Hash: btcHeader.Hash()}, serializedStakingTx, blockWithStakingTx.SpvProof.MerkleNodes)

	slashignSpendInfo, err := testStakingInfo.StakingInfo.SlashingPathSpendInfo()
	require.NoError(t, err)

	// delegator sig
	delegatorSig, err := testStakingInfo.SlashingTx.Sign(
		testStakingInfo.StakingTx,
		0,
		slashignSpendInfo.GetPkScriptPath(),
		delBtcPrivKey,
	)
	require.NoError(t, err)

	unbondingValue := stakingAmount - 1000
	stakingTxHash := testStakingInfo.StakingTx.TxHash()

	testUnbondingInfo := datagen.GenBTCUnbondingSlashingInfo(
		r,
		t,
		e2etest.BtcNetworkParams,
		delBtcPrivKey,
		fpPks,
		params.CovenantPks,
		params.CovenantQuorum,
		wire.NewOutPoint(&stakingTxHash, 0),
		unbondingTime,
		unbondingValue,
		params.SlashingAddress.String(),
		params.SlashingRate,
		unbondingTime,
	)

	unbondingTxMsg := testUnbondingInfo.UnbondingTx

	unbondingSlashingPathInfo, err := testUnbondingInfo.UnbondingInfo.SlashingPathSpendInfo()
	require.NoError(t, err)

	unbondingSig, err := testUnbondingInfo.SlashingTx.Sign(
		unbondingTxMsg,
		0,
		unbondingSlashingPathInfo.GetPkScriptPath(),
		delBtcPrivKey,
	)
	require.NoError(t, err)

	serializedUnbondingTx, err := bbntypes.SerializeBTCTx(testUnbondingInfo.UnbondingTx)
	require.NoError(t, err)

	// submit the BTC delegation to Babylon
	_, err = stm.BBNClient.CreateBTCDelegation(
		delBabylonPubKey.(*secp256k1.PubKey),
		bbntypes.NewBIP340PubKeyFromBTCPK(delBtcPubKey),
		fpPks,
		pop,
		uint32(stakingTime),
		stakingAmount,
		txInfo,
		testStakingInfo.SlashingTx,
		delegatorSig,
		serializedUnbondingTx,
		uint32(unbondingTime),
		unbondingValue,
		testUnbondingInfo.SlashingTx,
		unbondingSig)
	require.NoError(t, err)

	t.Log("successfully submitted a BTC delegation")

	return &e2etestbnb.TestDelegationData{
		DelegatorPrivKey:        delBtcPrivKey,
		DelegatorKey:            delBtcPubKey,
		DelegatorBabylonPrivKey: delBabylonPrivKey.(*secp256k1.PrivKey),
		DelegatorBabylonKey:     delBabylonPubKey.(*secp256k1.PubKey),
		FpPks:                   fpPks,
		StakingTx:               testStakingInfo.StakingTx,
		SlashingTx:              testStakingInfo.SlashingTx,
		StakingTxInfo:           txInfo,
		DelegatorSig:            delegatorSig,
		SlashingAddr:            params.SlashingAddress.String(),
		StakingTime:             stakingTime,
		StakingAmount:           stakingAmount,
	}
}

func (stm *SdkTestManager) WaitForNPendingDels(t *testing.T, n int) []*bstypes.BTCDelegationResponse {
	var (
		dels []*bstypes.BTCDelegationResponse
		err  error
	)
	require.Eventually(t, func() bool {
		dels, err = stm.BBNClient.QueryPendingDelegations(
			100,
		)
		if err != nil {
			return false
		}
		return len(dels) == n
	}, e2etest.EventuallyWaitTimeOut, e2etest.EventuallyPollTime)

	t.Logf("delegations are pending")

	return dels
}

func (stm *SdkTestManager) WaitForNActiveDels(t *testing.T, n int) []*bstypes.BTCDelegationResponse {
	var (
		dels []*bstypes.BTCDelegationResponse
		err  error
	)
	require.Eventually(t, func() bool {
		dels, err = stm.BBNClient.QueryActiveDelegations(
			100,
		)
		if err != nil {
			return false
		}
		return len(dels) == n
	}, e2etest.EventuallyWaitTimeOut, e2etest.EventuallyPollTime)

	t.Logf("delegations are active")

	return dels
}

func (stm *SdkTestManager) InsertCovenantSigForDelegation(t *testing.T, btcDel *bstypes.BTCDelegation) {
	slashingTx := btcDel.SlashingTx
	stakingTx := btcDel.StakingTx
	stakingMsgTx, err := bbntypes.NewBTCTxFromBytes(stakingTx)
	require.NoError(t, err)

	params := stm.StakingParams

	var fpKeys []*btcec.PublicKey
	for _, v := range btcDel.FpBtcPkList {
		fpKeys = append(fpKeys, v.MustToBTCPK())
	}

	stakingInfo, err := btcstaking.BuildStakingInfo(
		btcDel.BtcPk.MustToBTCPK(),
		fpKeys,
		params.CovenantPks,
		params.CovenantQuorum,
		btcDel.GetStakingTime(),
		btcutil.Amount(btcDel.TotalSat),
		e2etest.BtcNetworkParams,
	)
	require.NoError(t, err)
	stakingTxUnbondingPathInfo, err := stakingInfo.UnbondingPathSpendInfo()
	require.NoError(t, err)

	idx, err := bbntypes.GetOutputIdxInBTCTx(stakingMsgTx, stakingInfo.StakingOutput)
	require.NoError(t, err)

	require.NoError(t, err)
	slashingPathInfo, err := stakingInfo.SlashingPathSpendInfo()
	require.NoError(t, err)

	var valEncKeys []*asig.EncryptionKey
	for _, v := range btcDel.FpBtcPkList {
		// get covenant private key from the keyring
		valEncKey, err := asig.NewEncryptionKeyFromBTCPK(v.MustToBTCPK())
		require.NoError(t, err)
		valEncKeys = append(valEncKeys, valEncKey)
	}

	unbondingMsgTx, err := bbntypes.NewBTCTxFromBytes(btcDel.BtcUndelegation.UnbondingTx)
	require.NoError(t, err)
	unbondingInfo, err := btcstaking.BuildUnbondingInfo(
		btcDel.BtcPk.MustToBTCPK(),
		fpKeys,
		params.CovenantPks,
		params.CovenantQuorum,
		uint16(btcDel.UnbondingTime),
		btcutil.Amount(unbondingMsgTx.TxOut[0].Value),
		e2etest.BtcNetworkParams,
	)
	require.NoError(t, err)

	var covenantAdaptorStakingSlashing1List [][]byte
	for _, v := range valEncKeys {
		// Covenant 0 signatures
		covenantAdaptorStakingSlashing1, err := slashingTx.EncSign(
			stakingMsgTx,
			idx,
			slashingPathInfo.RevealedLeaf.Script,
			stm.CovenantPrivKeys[0],
			v,
		)
		require.NoError(t, err)
		covenantAdaptorStakingSlashing1List = append(covenantAdaptorStakingSlashing1List, covenantAdaptorStakingSlashing1.MustMarshal())
	}

	covenantUnbondingSig1, err := btcstaking.SignTxWithOneScriptSpendInputFromTapLeaf(
		unbondingMsgTx,
		stakingInfo.StakingOutput,
		stm.CovenantPrivKeys[0],
		stakingTxUnbondingPathInfo.RevealedLeaf,
	)
	require.NoError(t, err)

	var covenantAdaptorUnbondingSlashing1List [][]byte
	for _, v := range valEncKeys {
		// slashing unbonding tx sig
		unbondingTxSlashingPathInfo, err := unbondingInfo.SlashingPathSpendInfo()
		require.NoError(t, err)
		covenantAdaptorUnbondingSlashing1, err := btcDel.BtcUndelegation.SlashingTx.EncSign(
			unbondingMsgTx,
			0,
			unbondingTxSlashingPathInfo.RevealedLeaf.Script,
			stm.CovenantPrivKeys[0],
			v,
		)
		require.NoError(t, err)
		covenantAdaptorUnbondingSlashing1List = append(covenantAdaptorUnbondingSlashing1List, covenantAdaptorUnbondingSlashing1.MustMarshal())
	}

	_, err = stm.BBNClient.SubmitCovenantSigs(
		stm.CovenantPrivKeys[0].PubKey(),
		stakingMsgTx.TxHash().String(),
		covenantAdaptorStakingSlashing1List,
		covenantUnbondingSig1,
		covenantAdaptorUnbondingSlashing1List,
	)
	require.NoError(t, err)

	var covenantAdaptorStakingSlashing2List [][]byte
	for _, v := range valEncKeys {
		// Covenant 1 signatures
		covenantAdaptorStakingSlashing2, err := slashingTx.EncSign(
			stakingMsgTx,
			idx,
			slashingPathInfo.RevealedLeaf.Script,
			stm.CovenantPrivKeys[1],
			v,
		)
		require.NoError(t, err)
		covenantAdaptorStakingSlashing2List = append(covenantAdaptorStakingSlashing2List, covenantAdaptorStakingSlashing2.MustMarshal())
	}

	covenantUnbondingSig2, err := btcstaking.SignTxWithOneScriptSpendInputFromTapLeaf(
		unbondingMsgTx,
		stakingInfo.StakingOutput,
		stm.CovenantPrivKeys[1],
		stakingTxUnbondingPathInfo.RevealedLeaf,
	)
	require.NoError(t, err)

	var covenantAdaptorUnbondingSlashing2List [][]byte
	for _, v := range valEncKeys {
		// slashing unbonding tx sig
		unbondingTxSlashingPathInfo, err := unbondingInfo.SlashingPathSpendInfo()
		require.NoError(t, err)
		covenantAdaptorUnbondingSlashing2, err := btcDel.BtcUndelegation.SlashingTx.EncSign(
			unbondingMsgTx,
			0,
			unbondingTxSlashingPathInfo.RevealedLeaf.Script,
			stm.CovenantPrivKeys[1],
			v,
		)
		require.NoError(t, err)
		covenantAdaptorUnbondingSlashing2List = append(covenantAdaptorUnbondingSlashing2List, covenantAdaptorUnbondingSlashing2.MustMarshal())
	}

	_, err = stm.BBNClient.SubmitCovenantSigs(
		stm.CovenantPrivKeys[1].PubKey(),
		stakingMsgTx.TxHash().String(),
		covenantAdaptorStakingSlashing2List,
		covenantUnbondingSig2,
		covenantAdaptorUnbondingSlashing2List,
	)
	require.NoError(t, err)
}

func (stm *SdkTestManager) Stop(t *testing.T) {
	stm.OpConsumerTestManager.Stop(t)
}
