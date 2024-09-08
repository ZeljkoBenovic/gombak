package main

import (
	"fmt"
	"os"

	"github.com/ZeljkoBenovic/gombak/internal/app"
	"github.com/ZeljkoBenovic/gombak/pkg/config"
	"github.com/ZeljkoBenovic/gombak/pkg/logger"
	"github.com/ZeljkoBenovic/gombak/pkg/service"
)

func main() {
	conf := config.NewConfig()

	log, err := logger.New(conf)
	if err != nil {
		fmt.Printf("could not create new logger: %s", err.Error())
		os.Exit(1)
	}

	run := app.NewApp(conf, log).AppModeFactory()

	srv, err := service.New(conf, []string{"run", "-c", conf.ConfigFilePath}, log)
	if err != nil {
		log.Info("could not init new service", "err", err)
		os.Exit(1)
	}

	err, isService := srv.HandleServiceCLICommands(run)
	if err != nil {
		log.Error("service error", "err", err)

		os.Exit(1)
	}

	if !isService {
		if err = run(); err != nil {
			log.Error("run error", "err", err)

			os.Exit(1)
		}
	}
}
