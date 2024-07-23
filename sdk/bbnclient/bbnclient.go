package bbnclient

import (
	"github.com/babylonchain/babylon/client/query"
	btcstakingtypes "github.com/babylonchain/babylon/x/btcstaking/types"
	sdkquerytypes "github.com/cosmos/cosmos-sdk/types/query"
)

type Client struct {
	*query.QueryClient
}

func (bbnClient *Client) QueryAllFpBtcPubKeys(consumerId string) ([]string, error) {
	pagination := &sdkquerytypes.PageRequest{}
	resp, err := bbnClient.QueryClient.QueryConsumerFinalityProviders(consumerId, pagination)
	if err != nil {
		return nil, err
	}

	var pkArr []string

	for _, fp := range resp.FinalityProviders {
		pkArr = append(pkArr, fp.BtcPk.MarshalHex())
	}
	return pkArr, nil
}

func (bbnClient *Client) QueryFpPower(fpPubkeyHex string, btcHeight uint64) (uint64, error) {
	totalPower := uint64(0)
	pagination := &sdkquerytypes.PageRequest{}
	// queries the BTCStaking module for all delegations of a finality provider
	resp, err := bbnClient.QueryClient.FinalityProviderDelegations(fpPubkeyHex, pagination)
	if err != nil {
		return 0, err
	}
	for {
		// btcDels contains all the queried BTC delegations
		for _, btcDels := range resp.BtcDelegatorDelegations {
			for _, btcDel := range btcDels.Dels {
				// check whether the delegation is active
				isActive, err := bbnClient.isDelegationActive(btcDel, btcHeight)
				if err != nil {
					return 0, err
				}
				if isActive {
					totalPower += btcDel.TotalSat
				}
			}
		}
		if resp.Pagination == nil || resp.Pagination.NextKey == nil {
			break
		}
		pagination.Key = resp.Pagination.NextKey
	}

	return totalPower, nil
}

func (bbnClient *Client) QueryMultiFpPower(
	fpPubkeyHexList []string,
	btcHeight uint64,
) (map[string]uint64, error) {
	fpPowerMap := make(map[string]uint64)

	for _, fpPubkeyHex := range fpPubkeyHexList {
		fpPower, err := bbnClient.QueryFpPower(fpPubkeyHex, btcHeight)
		if err != nil {
			return nil, err
		}
		fpPowerMap[fpPubkeyHex] = fpPower
	}

	return fpPowerMap, nil
}

func (bbnClient *Client) QueryFpEarliestDelHeight(fpPubkeyHex string) (*uint64, error) {
	pagination := &sdkquerytypes.PageRequest{
		Limit: 100,
	}
	// queries the BTCStaking module for all delegations of a finality provider
	resp, err := bbnClient.QueryClient.FinalityProviderDelegations(fpPubkeyHex, pagination)
	if err != nil {
		return nil, err
	}
	var earliestBtcHeight *uint64
	for {
		// btcDels contains all the queried BTC delegations
		for _, btcDels := range resp.BtcDelegatorDelegations {
			for _, btcDel := range btcDels.Dels {
				btcDelHeight, err := bbnClient.getDelConfirmHeight(btcDel)
				if err != nil {
					continue
				}
				if btcDelHeight != nil && (earliestBtcHeight == nil || *btcDelHeight < *earliestBtcHeight) {
					earliestBtcHeight = btcDelHeight
				}
			}
		}
		if resp.Pagination == nil || resp.Pagination.NextKey == nil {
			break
		}
		pagination.Key = resp.Pagination.NextKey
	}
	return earliestBtcHeight, nil
}

func (bbnClient *Client) getDelConfirmHeight(
	btcDel *btcstakingtypes.BTCDelegationResponse,
) (*uint64, error) {
	btccheckpointParams, err := bbnClient.QueryClient.BTCCheckpointParams()
	if err != nil {
		return nil, err
	}
	btcstakingParams, err := bbnClient.QueryClient.BTCStakingParams()
	if err != nil {
		return nil, err
	}
	kValue := btccheckpointParams.GetParams().BtcConfirmationDepth
	covQuorum := btcstakingParams.GetParams().CovenantQuorum

	btcHeader, err := bbnClient.QueryClient.BTCHeaderChainTip()
	if err != nil {
		return nil, err
	}
	latestBtcHeight := btcHeader.Header.Height
	confirmationHeight := btcDel.StartHeight + kValue
	// The active delegation needs to satisfy:
	// 1) the staking tx is k-deep in Bitcoin, i.e., start_height + k
	// 2) it receives a quorum number of covenant committee signatures
	if latestBtcHeight > confirmationHeight && btcDel.EndHeight > confirmationHeight && uint32(len(btcDel.CovenantSigs)) > covQuorum {
		return &confirmationHeight, nil
	}
	return nil, nil
}
