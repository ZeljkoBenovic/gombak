package discovery

import (
	"fmt"

	"github.com/ZeljkoBenovic/gombak/pkg/discovery/l2tp"
	"github.com/ZeljkoBenovic/gombak/pkg/logger"
)

type Discovery interface {
	GetIPAddresses() ([]string, error)
}

type Config struct {
	APIPort    string
	APISSLPort string

	Hosts    []string
	Username string
	Password string

	Log *logger.Logger
}

type DiscConfigFn func(config *Config) (Discovery, error)

type Type string

const (
	L2TP Type = "l2tp"
)

var Discoverers = map[Type]DiscConfigFn{
	L2TP: func(c *Config) (Discovery, error) {
		if c.Hosts == nil {
			return nil, fmt.Errorf("hosts not found")
		}

		return l2tp.NewL2TP(c.Hosts, c.APIPort, c.APISSLPort, c.Username, c.Password, c.Log), nil
	},
}
