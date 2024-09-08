package app

import (
	"fmt"
	"sync"

	"github.com/ZeljkoBenovic/gombak/pkg/backup"
	"github.com/ZeljkoBenovic/gombak/pkg/config"
	"github.com/ZeljkoBenovic/gombak/pkg/discovery"
	"github.com/ZeljkoBenovic/gombak/pkg/logger"
)

type App struct {
	conf config.Config
	log  *logger.Logger
	wg   *sync.WaitGroup

	routersDone *routersDone
}

type routersDone struct {
	done map[string]struct{}
	mut  *sync.RWMutex
}

func (r *routersDone) isDone(routerName string) bool {
	r.mut.RLock()
	if _, ok := r.done[routerName]; ok {
		r.mut.RUnlock()
		return true
	}
	r.mut.RUnlock()

	r.mut.Lock()
	r.done[routerName] = struct{}{}
	r.mut.Unlock()

	return false
}

func NewApp(conf config.Config, log *logger.Logger) App {
	return App{
		conf: conf,
		log:  log,
		wg:   &sync.WaitGroup{},
		routersDone: &routersDone{
			done: make(map[string]struct{}),
			mut:  &sync.RWMutex{},
		},
	}
}

// AppModeFactory returns a worker function based on the configured mode
func (a App) AppModeFactory() func() error {
	switch a.conf.Mode {
	case config.SingleRouter:
		return func() error {
			a.log.Info("Running single mode router backup...")

			if err := a.conf.CheckSingleRequirements(); err != nil {
				return err
			}

			if err := a.singleRouterBackup(
				a.conf.Single.Host,
				a.conf.Single.Port,
				a.conf.Single.Username,
				a.conf.Single.Password,
			); err != nil {
				return err
			}

			a.log.Info("Single router backup complete")

			return backup.RunFileCleanup(a.conf.BackupFolder, a.conf.BackupRetentionDays, a.log)
		}
	case config.MultiRouter:
		return func() error {
			a.log.Info("Running multi router backup mode...")

			for _, mt := range a.conf.Multi {
				mt := mt

				a.wg.Add(1)

				go func() {
					defer a.wg.Done()

					if err := a.singleRouterBackup(mt.Host, mt.Port, mt.Username, mt.Password); err != nil {
						a.log.Error("Could not perform backup", "err", err.Error(), "host", mt.Host)

						return
					}
				}()
			}

			a.wg.Wait()

			a.log.Info("Multi router backup complete")

			return backup.RunFileCleanup(a.conf.BackupFolder, a.conf.BackupRetentionDays, a.log)
		}
	case config.L2TPDiscovery:
		return func() error {
			a.log.Info("Running l2tp discovery mode...")

			if err := a.conf.CheckDiscoveryRequirements(); err != nil {
				return fmt.Errorf("discovery mode requirements not met: %w", err)
			}

			disc, err := discovery.Discoverers[discovery.L2TP](&discovery.Config{
				APIPort:    a.conf.Discovery.APIPort,
				APISSLPort: a.conf.Discovery.APISSLPort,
				Hosts:      a.conf.Discovery.Hosts,
				Username:   a.conf.Discovery.Username,
				Password:   a.conf.Discovery.Password,
				Log:        a.log,
			})
			if err != nil {
				return err
			}

			discRouters, err := disc.GetIPAddresses()
			if err != nil {
				return err
			}

			for name, ip := range discRouters {
				a.wg.Add(1)

				ip := ip
				name := name

				go func() {
					defer a.wg.Done()

					if err := a.singleRouterBackup(
						ip,
						a.conf.Discovery.SSHPort,
						a.conf.Discovery.Username,
						a.conf.Discovery.Password,
					); err != nil {
						a.log.Error("Could not perform backup", "err", err.Error(), "host", name, "ip", ip)
						return
					}
				}()
			}

			a.wg.Wait()

			a.log.Info("Discovery mode routers backup complete")

			return backup.RunFileCleanup(a.conf.BackupFolder, a.conf.BackupRetentionDays, a.log)
		}
	default:
		return func() error {
			return fmt.Errorf("mode not supported")
		}
	}
}

func (a App) singleRouterBackup(host, port, user, pass string) error {
	bck, err := backup.New(
		host,
		port,
		user,
		pass,
		a.log,
	)
	if err != nil {
		return err
	}

	defer bck.Close()

	routerName, err := bck.GetRouterIdentity()
	if err != nil {
		return err
	}

	// Skip work if already done
	if a.routersDone.isDone(routerName) {
		return nil
	}

	if err := bck.RunBackup(a.conf.BackupFolder); err != nil {
		return err
	}

	return bck.DeleteTempFiles()
}
