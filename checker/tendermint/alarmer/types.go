package alarmer

import "tendermint-checker/types"

type Alert struct {
	alarmer types.Alarmer
}

func NewAlert(alarmer types.Alarmer) Alert {
	return Alert{
		alarmer: alarmer,
	}
}
