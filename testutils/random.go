package testutils

import (
	"math/rand"

	"github.com/babylonchain/babylon-finality-gadget/sdk/cwclient"
	"github.com/ethereum/go-ethereum/common"
)

func RandomHash(rng *rand.Rand) (out common.Hash) {
	rng.Read(out[:])
	return
}

func RandomL2Block(rng *rand.Rand) (out cwclient.L2Block) {
	out.BlockHash = RandomHash(rng).String()
	out.BlockHeight = rng.Uint64()
	out.BlockTimestamp = rng.Uint64()
	return
}
