package cwclient

type CosmWasmClientInterface interface {
	QueryListOfVotedFinalityProviders(queryParams *L2Block) ([]string, error)
	QueryConsumerId() (string, error)
	QueryIsEnabled() (bool, error)
}
