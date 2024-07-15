package client

import "github.com/babylonchain/babylon-finality-gadget/sdk/cwclient"

type SdkClientInterface interface {
	QueryIsBlockBabylonFinalized(queryParams *cwclient.L2Block) (bool, error)

	QueryBlockRangeBabylonFinalized(queryBlocks []*cwclient.L2Block) (*uint64, error)
}
