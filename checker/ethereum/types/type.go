package types

type Checker interface {
	Run(c *CheckerConfig, client *CheckerClient)
}

type Func func(c *CheckerConfig, client *CheckerClient)

func (f Func) Run(c *CheckerConfig, client *CheckerClient) {
	f(c, client)
}
