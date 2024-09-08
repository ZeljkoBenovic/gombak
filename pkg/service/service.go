package service

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ZeljkoBenovic/gombak/pkg/config"
	"github.com/ZeljkoBenovic/gombak/pkg/logger"
	srv "github.com/kardianos/service"
)

type Service struct {
	svc srv.Service
	log *logger.Logger

	runner *serviceRunner
}

type serviceRunner struct {
	runFn  func() error
	log    srv.Logger
	stopCh chan struct{}
	conf   config.Config
}

func New(conf config.Config, args []string, log *logger.Logger) (*Service, error) {
	s := &Service{
		log: log,
		runner: &serviceRunner{
			stopCh: make(chan struct{}),
			conf:   conf,
		},
	}

	srvc, err := srv.New(s.runner, &srv.Config{
		Name:        "GoMBak",
		DisplayName: "GoMBak",
		Description: "Provides a Mikrotik router backup service. More info: https://github.com/zeljkobenovic/gombak",
		Arguments:   args,
	})
	if err != nil {
		return nil, fmt.Errorf("could not init service: %w", err)
	}

	lgr, err := srvc.Logger(nil)
	if err != nil {
		return nil, fmt.Errorf("could not create service logger: %w", err)
	}

	s.svc = srvc
	s.runner.log = lgr

	return s, nil
}

// HandleServiceCLICommands will handle "install", "uninstall" and "run" cli commands which handle gombak as a system service.
// If these cli arguments are not set, this method returns false signaling that it should be run as a console program.
func (s *Service) HandleServiceCLICommands(runFn func() error) (err error, isService bool) {
	isService = true
	err = nil

	switch os.Args[1] {
	case "install":
		if err := srv.Control(s.svc, "install"); err != nil {
			return fmt.Errorf("could not install gombak service: %w", err), isService
		}

		if err := srv.Control(s.svc, "start"); err != nil {
			return fmt.Errorf("could not start gombak service: %w", err), isService
		}

		return
	case "uninstall":
		if err := srv.Control(s.svc, "uninstall"); err != nil {
			return fmt.Errorf("could not uninstall gombak service: %w", err), isService
		}

		return
	case "run":
		s.runner.runFn = runFn
		err = s.svc.Run()

		return
	}

	return nil, false
}

func (s *serviceRunner) Start(_ srv.Service) error {
	if s.runFn == nil {
		return fmt.Errorf("runFn function not initialized")
	}

	go func() {
		ticker := time.NewTicker(time.Hour * 24 * time.Duration(s.conf.BackupFrequencyDays))
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				_ = s.log.Info("running mikrotik backup per schedule")
				if err := s.runFn(); err != nil {
					if err := s.log.Error(err); err != nil {
						log.Println(err)
					}
				}
			case <-s.stopCh:
				_ = s.log.Info("stopping gombak service")

				return
			}
		}
	}()

	return nil
}

func (s *serviceRunner) Stop(_ srv.Service) error {
	s.stopCh <- struct{}{}
	return nil
}
