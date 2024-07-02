package main

import (
	"fmt"

	"github.com/babylonchain/babylon-da-sdk/sdk"
	"github.com/babylonchain/babylon-da-sdk/sdk/btc"
)

func checkBlockFinalized(height uint64, hash string) {
	client, err := sdk.NewClient(&sdk.Config{
		ChainType: 0,
		// TODO: avoid using stub contract
		ContractAddr: "bbn1ghd753shjuwexxywmgs4xz7x2q732vcnkm6h2pyv9s6ah3hylvrqxxvh0f",
		BitcoinRpc:   btc.RpcURL,
	})

	if err != nil {
		fmt.Printf("error creating client: %v\n", err)
		return
	}

	isFinalized, err := client.QueryIsBlockBabylonFinalized(&sdk.L2Block{
		BlockHeight:    height,
		BlockHash:      hash,
		BlockTimestamp: uint64(1718332131),
	})
	if err != nil {
		if _, ok := err.(*sdk.NoFpHasVotingPowerError); ok {
			fmt.Printf("checking block %d: no FP has voting power, skip it\n", height)
		}
	} else {
		fmt.Printf("is block %d finalized?: %t\n", height, isFinalized)
	}
}

func main() {
	// TODO: this will always return false. we should find a better way to demo it
	checkBlockFinalized(uint64(2), "0x1000000000000000000000000000000000000000000000000000000000000000")
}
