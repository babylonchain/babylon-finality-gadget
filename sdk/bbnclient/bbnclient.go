package bbnclient

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/babylonchain/babylon/client/query"
	"github.com/babylonchain/babylon/x/btcstaking/types"
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

func (bbnClient *Client) QueryEarliestActiveDelBtcHeight(fpPkHexList []string) (*uint64, error) {
	var activeDelHeight atomic.Value
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var wg sync.WaitGroup
	errChan := make(chan error, 1)

	// find the earliest BTC delegation height among all FP delegations
	for _, fpPkHex := range fpPkHexList {
		wg.Add(1)
		go func(fpPkHex string) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			default:
				fpActiveDelHeight, err := bbnClient.QueryFpEarliestActiveDelBtcHeight(fpPkHex)
				if err != nil {
					cancel()
					select {
					case errChan <- err:
					default:
					}
					return
				}
				if fpActiveDelHeight != nil {
					for {
						current := activeDelHeight.Load()
						if current == nil || *fpActiveDelHeight < *current.(*uint64) {
							if activeDelHeight.CompareAndSwap(current, fpActiveDelHeight) {
								break
							}
						} else {
							break
						}
					}
				}
			}
		}(fpPkHex)
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

	if err := <-errChan; err != nil {
		return nil, err
	}

	if val := activeDelHeight.Load(); val != nil {
		return val.(*uint64), nil
	}
	return nil, nil
}

func (bbnClient *Client) QueryFpEarliestActiveDelBtcHeight(fpPubkeyHex string) (*uint64, error) {
	pagination := &sdkquerytypes.PageRequest{
		Limit: 100,
	}
	// queries the BTCStaking module for all delegations of a finality provider
	resp, err := bbnClient.QueryClient.FinalityProviderDelegations(fpPubkeyHex, pagination)
	if err != nil {
		return nil, err
	}
	// queries BtcConfirmationDepth, CovenantQuorum, and the latest BTC header
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
	var earliestBtcHeight *uint64
	for {
		// btcDels contains all the queried BTC delegations
		for _, btcDels := range resp.BtcDelegatorDelegations {
			for _, btcDel := range btcDels.Dels {
				// check whether the delegation is active
				latestBtcHeight := btcHeader.GetHeader().Height
				confirmationHeight := btcDel.StartHeight + kValue
				if isActiveBtcDelegation(btcDel, latestBtcHeight, confirmationHeight, covQuorum) {
					if earliestBtcHeight == nil || confirmationHeight < *earliestBtcHeight {
						earliestBtcHeight = &confirmationHeight
					}
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

// The active delegation needs to satisfy:
// 1) the staking tx is k-deep in Bitcoin, i.e., start_height + k
// 2) it receives a quorum number of covenant committee signatures
func isActiveBtcDelegation(btcDel *types.BTCDelegationResponse, latestBtcHeight, confirmationHeight uint64, covQuorum uint32) bool {
	return latestBtcHeight > confirmationHeight &&
		btcDel.EndHeight > confirmationHeight &&
		uint32(len(btcDel.CovenantSigs)) > covQuorum
}
