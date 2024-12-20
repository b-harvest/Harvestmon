package main

import (
	"flag"
	"github.com/b-harvest/Harvestmon/repository"
	log "github.com/sirupsen/logrus"
	"os"
	"time"
)

var (
	cfg *Config
)

func init() {
	cfg = &Config{}
	time.Local = time.UTC

	var (
		logLevel string
		l        log.Level
		err      error
	)

	err = initializeViper()
	if err != nil {
		panic(err)
	}

	flag.StringVar(&logLevel, "log-level", "info", "allow showing debug log")
	flag.Parse()
	if os.Getenv("LOG_LEVEL") != "" {
		logLevel = os.Getenv("LOG_LEVEL")
	}

	l, err = log.ParseLevel(logLevel)
	if err != nil {
		panic(err)
	}
	log.SetLevel(l)
	log.SetFormatter(&log.TextFormatter{})

	configPath := "./config.toml"
	if os.Getenv("CONFIG_PATH") != "" {
		configPath = os.Getenv("CONFIG_PATH")
	}
	cfg.logger = log.NewEntry(log.StandardLogger())

	if err = loadConfig(configPath); err != nil {
		panic(err)
	}

}

func main() {
	var (
		storeQueue = make(chan repository.StoreEntity)
	)

	defer func() {
		close(storeQueue)
	}()

	go cfg.Store.StartWithStoreQueue(storeQueue)

	cfg.logger.Infof("starting monitor: %v", cfg.AgentName)
	for _, m := range cfg.getMonitorConfigs() {
		for _, collector := range m.getCollectors() {
			collector.logger.Infof("starting collector")
			go func(collector Collector) {
				collector.CollectorFunc(m)(storeQueue, collector.logger)
				t := time.NewTicker(*collector.Interval)
				defer t.Stop()
				for {
					select {
					case <-cfg.ctx.Done():
						return
					case <-t.C:
						collector.CollectorFunc(m)(storeQueue, collector.logger)
					}
				}
			}(collector)
		}
	}

	saved := make(chan interface{})
	cfg.SaveOnExit(saved)
	<-cfg.ctx.Done()
	<-saved
}
