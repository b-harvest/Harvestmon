package alarmer

import (
	"errors"
	"fmt"
	"github.com/b-harvest/Harvestmon/log"
)

func RunAlarm(alert Alert) {

	log.Error(errors.New(fmt.Sprintf("%s,%s", alert.alarmer.AlarmerName, alert.alarmer.Image)))

}
