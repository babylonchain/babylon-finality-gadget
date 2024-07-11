package sdk

import "fmt"

var (
	ErrNoFpHasVotingPower = fmt.Errorf("no FP has voting power for the consumer chain")
	ErrEmptyQueryBlocks   = fmt.Errorf("query blocks is empty")
)
