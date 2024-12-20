package main

import (
	"flag"
	log "github.com/sirupsen/logrus"
	"os"
	"sync"
	"time"
)

var (
	configManager *ConfigManager
)

func init() {
	time.Local = time.UTC

	var (
		logLevel   string
		configFile string
		l          log.Level
		err        error
	)

	err = InitializeViper()
	if err != nil {
		panic(err)
	}

	flag.StringVar(&logLevel, "log-level", "", "allow showing debug log")
	flag.StringVar(&configFile, "config", "", "configuration file path")
	flag.Parse()
	if logLevel == "" && os.Getenv("LOG_LEVEL") != "" {
		logLevel = os.Getenv("LOG_LEVEL")
	} else {
		logLevel = "info"
	}

	if configFile == "" {
		if os.Getenv("CONFIG_PATH") != "" {
			configFile = os.Getenv("CONFIG_PATH")
		} else {
			configFile = "./config.toml"
		}
	}

	l, err = log.ParseLevel(logLevel)
	if err != nil {
		panic(err)
	}
	log.SetLevel(l)

	configManager = &ConfigManager{}
	configManager.logger = log.NewEntry(log.StandardLogger())

	if err = configManager.LoadConfig(configFile); err != nil {
		configManager.logger.Fatalf("Error loading configuration: %v", err)
		panic(err)
	}

	configManager.config.GithubFile.isChanged()

}

func main() {
	defer func() {
		if db, _ := configManager.config.writeRepo.DB.DB(); db.Close() != nil {
			configManager.logger.Error("Error closing writeRepository db")
		}
		if db, _ := configManager.config.readRepo.DB.DB(); db.Close() != nil {
			configManager.logger.Error("Error closing readRepository db")
		}
	}()

	start()
}

func handleAction() error {
	cc, err := configManager.GetConfig()
	if err != nil {
		configManager.logger.Fatalf("Error loading configuration: %v", err)
		return nil
	}

	cc.logger.Infof("starting hanlder")
	defer cc.logger.Infof("finishing hanlder")

	var (
		wg        sync.WaitGroup
		semaphore = make(chan struct{}, 50)
	)

	resolvTs := time.Now()
	for _, nodeInfo := range cc.NodeInfos {
		semaphore <- struct{}{}
		wg.Add(1)
		go func(n NodeInfo) {
			defer func() {
				<-semaphore
				if r := recover(); r != nil {
					cc.logger.Errorf("Recovered from panic: %v. node: %v", r, n.Name)
				}
				wg.Done()
			}()

			err = n.loadAlerts(cc.readRepo)
			if err != nil {
				cc.logger.Fatalf("Error loading alerts: %v", err)
			}

			if n.Alerts.isSnooze() {
				// snooze
				return
			}
			for targetName, target := range n.Alerts.Ethereum {
				if target.isSnooze() {
					// snooze
					continue
				}
				for _, strategy := range target.getAlertStrategies() {
					event, msg, shouldAlert := strategy.check(n.Name)

					if !shouldAlert {
						target.logger.Infof("target[%10s] is healthy", strategy.getName())
						if msg != "" {
							n.alert(cc.readRepo, targetName, msg, strategy.getAlertLevel(), event, true, nil)
						} else {

							// if there is empty msg, ignore it.
						}
					} else {

						// if `isAlert` is true but no msg, it means there may be occurred error while checking.
						if msg == "" {
							n.logger.Debugf("skipping alert %s[%s] - %s", n.Name, targetName, strategy)
						} else {
							n.alert(cc.readRepo, targetName, msg, strategy.getAlertLevel(), event, false, nil)
						}

					}

				}
			}

			for targetName, target := range n.Alerts.Tendermint {
				if target.isSnooze() {
					// snooze
					continue
				}
				for _, strategy := range target.getAlertStrategies() {
					event, msg, shouldAlert := strategy.check(n.Name)
					if !shouldAlert {
						if msg != "" {
							target.logger.Infof("target[%10s] is healthy", strategy.getName())
							n.alert(cc.readRepo, targetName, msg, strategy.getAlertLevel(), event, true, nil)
						} else {
							// if there is empty msg, ignore it.
						}
					} else {

						// if `isAlert` is true but no msg, it means there may be occurred error while checking.
						if msg == "" {
							n.logger.Debugf("skipping alert %s[%s] - %s", n.Name, targetName, strategy)
						} else {
							n.alert(cc.readRepo, targetName, msg, strategy.getAlertLevel(), event, false, nil)
						}

					}

				}
			}

			err = n.saveAlerts(*cc.writeRepo, cc.CommitId, resolvTs)
			if err != nil {
				cc.logger.Fatalf("Error saving alerts: %v", err)
			}
		}(nodeInfo)

	}
	wg.Wait()

	return nil
}
