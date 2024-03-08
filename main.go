package main

import (
	"fmt"
	"os"

	"github.com/ZeljkoBenovic/gombak/internal/app"
	"github.com/ZeljkoBenovic/gombak/pkg/config"
	"github.com/ZeljkoBenovic/gombak/pkg/logger"
)

func main() {
	conf := config.NewConfig()

	log, err := logger.New(conf)
	if err != nil {
		fmt.Printf("Could not create new logger: %s", err.Error())
		os.Exit(1)
	}

	run := app.NewApp(conf, log).AppModeFactory()

	if err = run(); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}
