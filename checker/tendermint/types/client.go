package types

func NewCheckerClient(cfg *CheckerConfig) *CheckerClient {
	rpcClient := CheckerClient{
		DB: getDatabase(cfg),
	}
	return &rpcClient
}
