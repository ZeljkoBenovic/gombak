package l2tp

import (
	"crypto/tls"
	"fmt"
	"sync"

	"github.com/ZeljkoBenovic/gombak/pkg/logger"
	"github.com/go-routeros/routeros"
)

type L2TP struct {
	hosts      []string
	apiPort    string
	apiSslPort string
	user       string
	pass       string

	useSSLApi     bool
	sslSkipVerify bool

	log *logger.Logger
	wg  *sync.WaitGroup
	mut *sync.Mutex
}

type Opts func(*L2TP)

func WithUseSSLApi() Opts {
	return func(tp *L2TP) {
		tp.useSSLApi = true
	}
}

func WithSSLSkipVerify() Opts {
	return func(tp *L2TP) {
		tp.sslSkipVerify = true
	}
}

func NewL2TP(hosts []string, apiPort, apiSSLPort, user, pass string, log *logger.Logger, opts ...Opts) *L2TP {
	l := &L2TP{
		hosts:      hosts,
		apiPort:    apiPort,
		apiSslPort: apiSSLPort,
		user:       user,
		pass:       pass,

		useSSLApi:     false,
		sslSkipVerify: false,

		log: log,
		wg:  &sync.WaitGroup{},
		mut: &sync.Mutex{},
	}

	for _, f := range opts {
		f(l)
	}

	return l
}

func (l *L2TP) GetIPAddresses() ([]string, error) {
	var (
		resp []string
	)

	for _, h := range l.hosts {
		h := h

		l.wg.Add(1)

		l.log.Info("Discovering ips on host", "host", h)

		go func() {
			defer l.wg.Done()

			ips, err := l.fetchRouterIPs(h)
			if err != nil {
				l.log.Error("Could not discover ips", "err", err.Error())
			}

			l.mut.Lock()
			resp = append(resp, ips...)
			l.mut.Unlock()
		}()
	}

	l.wg.Wait()

	l.log.Info("Discovery complete", "total", len(resp))

	return resp, nil
}

func (l *L2TP) fetchRouterIPs(host string) ([]string, error) {
	var (
		tunnelNames []string
		remoteIPs   []string
		cl          *routeros.Client
		err         error
	)

	if l.useSSLApi {
		cl, err = routeros.DialTLS(fmt.Sprintf("%s:%s", host, l.apiSslPort), l.user, l.pass, &tls.Config{
			InsecureSkipVerify: l.sslSkipVerify,
		})
		if err != nil {
			return nil, fmt.Errorf("could not dial router: %w", err)
		}
	} else {
		cl, err = routeros.Dial(fmt.Sprintf("%s:%s", host, l.apiPort), l.user, l.pass)
		if err != nil {
			return nil, fmt.Errorf("could not dial router: %w", err)
		}
	}

	res, err := cl.Run("/interface/l2tp-client/print", "?running=true")
	if err != nil {
		return nil, fmt.Errorf("could not run l2tp-client print command: %w", err)
	}

	for _, s := range res.Re {
		tunnelNames = append(tunnelNames, s.Map["name"])
	}

	res, err = cl.Run("/interface/l2tp-server/print", "?running=true")
	if err != nil {
		return nil, fmt.Errorf("could not run l2tp-server print command: %w", err)
	}

	for _, s := range res.Re {
		tunnelNames = append(tunnelNames, s.Map["name"])
	}

	for _, tun := range tunnelNames {
		res, err = cl.Run("/ip/address/print", fmt.Sprintf("?interface=%s", tun))
		if err != nil {
			return nil, fmt.Errorf("could not run ip address find: %w", err)
		}

		for _, r := range res.Re {
			remoteIPs = append(remoteIPs, r.Map["network"])
		}
	}

	return remoteIPs, nil
}
