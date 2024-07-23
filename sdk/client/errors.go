package client

import "fmt"

var (
	ErrBtcStakingNotActivated = fmt.Errorf("BTC staking is not activated for the consumer chain")
)
