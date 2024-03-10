package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	flag "github.com/spf13/pflag"
)

type Mode string

const (
	SingleRouter  Mode = "single"
	MultiRouter   Mode = "multi"
	L2TPDiscovery Mode = "l2tp"
)

var AvailableModes = map[string]Mode{
	"single": SingleRouter,
	"multi":  MultiRouter,
	"l2tp":   L2TPDiscovery,
}

type Config struct {
	Mode                Mode         `koanf:"mode"`
	BackupFolder        string       `koanf:"backup-dir"`
	BackupRetentionDays int          `koanf:"retention-days"`
	Single              RouterInfo   `koanf:"single"`
	Discovery           Discovery    `koanf:"discovery"`
	Multi               []RouterInfo `koanf:"multi-router"`

	Logger Log `koanf:"log"`
}

type RouterInfo struct {
	Host     string `koanf:"host"`
	Port     string `koanf:"ssh-port"`
	Username string `koanf:"username"`
	Password string `koanf:"password"`
}

type Discovery struct {
	Hosts      []string `koanf:"hosts"`
	Username   string   `koanf:"username"`
	Password   string   `koanf:"password"`
	APIPort    string   `koanf:"api-port"`
	APISSLPort string   `koanf:"api-ssl-port"`
	SSHPort    string   `konaf:"ssh-port"`
}

type Log struct {
	JSONOutput bool   `koanf:"json"`
	File       string `koanf:"file"`
	Level      string `koanf:"level"`
}

var (
	ErrSingleHostNotFound     = errors.New("single mode router ip not found")
	ErrSingleUserNotFound     = errors.New("single mode username not found")
	ErrSinglePasswordNotFound = errors.New("single mode password not found")

	ErrDiscoveryHostsNotFound = errors.New("discovery mode router ip addresses not found")
	ErrDiscoveryUserNotFound  = errors.New("discovery mode username not found")
	ErrDiscoveryPassNotFound  = errors.New("discovery mode password not found")
)

var k = koanf.New(".")

func NewConfig() Config {
	var (
		c        = Config{}
		confFile string
		mode     string
		mrList   []RouterInfo
	)

	f := flag.NewFlagSet("config", flag.ContinueOnError)
	f.Usage = func() {
		fmt.Println(f.FlagUsages())
		os.Exit(0)
	}

	f.StringVarP(&confFile, "config", "c", "", "configuration yaml file")
	f.StringVarP(&c.BackupFolder, "backup-dir", "b", "mt-backup", "mikrotik backup export directory")
	f.StringVarP(&mode, "mode", "m", "single", "mode of operation")
	f.IntVarP(&c.BackupRetentionDays, "retention-days", "r", 5, "days of retention")

	f.StringVarP(&c.Single.Host, "single.host", "", "", "the ip address of the router")
	f.StringVarP(&c.Single.Port, "single.ssh-port", "", "22", "the ssh port of the router")
	f.StringVarP(&c.Single.Username, "single.user", "", "", "the username for the router")
	f.StringVarP(&c.Single.Password, "single.pass", "", "", "the password for the username")

	f.BoolVarP(&c.Logger.JSONOutput, "log.json", "", false, "output logs in json format")
	f.StringVarP(&c.Logger.File, "log.file", "", "", "write logs to the specified file")
	f.StringVarP(&c.Logger.Level, "log.level", "", "info", "define log level")

	_ = f.Parse(os.Args[1:])

	// load config file if defined
	if confFile != "" {
		if err := k.Load(file.Provider(confFile), yaml.Parser()); err != nil {
			log.Fatalln("Could not load config from file:", err.Error())
		}
	}

	// load environment variables
	if err := k.Load(env.Provider("GOMBAK_", ".", func(s string) string {
		return strings.Replace(strings.ToLower(
			strings.TrimPrefix(s, "GOMBAK_")), "_", ".", -1)
	}), nil); err != nil {
		log.Fatalln("Could not setup environment variables: ", err.Error())
	}

	// load flags
	if err := k.Load(posflag.Provider(f, ".", k), nil); err != nil {
		log.Fatalln("Could not create config:", err.Error())
	}

	if _, ok := AvailableModes[k.String("mode")]; !ok {
		log.Fatalln("Selected mode not available")
	}

	mr, ok := k.Get("multi-router").([]any)
	if mr != nil && !ok {
		log.Fatalln("Could not cast multi-router to []RouterInfo")
	}

	if err := k.Unmarshal("multi-router", &mrList); err != nil {
		log.Fatalln("Could not unmarshal router list")
	}

	return Config{
		BackupFolder:        k.String("backup-dir"),
		BackupRetentionDays: k.Int("retention-days"),
		Mode:                AvailableModes[k.String("mode")],
		Single: RouterInfo{
			Host:     k.String("single.host"),
			Port:     k.String("single.ssh-port"),
			Username: k.String("single.user"),
			Password: k.String("single.pass"),
		},
		Multi: mrList,
		Discovery: Discovery{
			Hosts:    k.Strings("discovery.hosts"),
			Username: k.String("discovery.username"),
			Password: k.String("discovery.password"),
			APIPort:  k.String("discovery.api-port"),
			SSHPort:  k.String("discovery.ssh-port"),
		},
		Logger: Log{
			JSONOutput: k.Bool("log.json"),
			File:       k.String("log.file"),
			Level:      k.String("log.level"),
		},
	}
}

func (c *Config) CheckSingleRequirements() error {
	if c.Single.Host == "" {
		return ErrSingleHostNotFound
	}

	if c.Single.Port == "" {
		c.Single.Port = "22"
	}

	if c.Single.Username == "" {
		return ErrSingleUserNotFound
	}

	if c.Single.Password == "" {
		return ErrSinglePasswordNotFound
	}

	return nil
}

func (c *Config) CheckDiscoveryRequirements() error {
	if c.Discovery.Hosts == nil {
		return ErrDiscoveryHostsNotFound
	}

	if c.Discovery.Username == "" {
		return ErrDiscoveryUserNotFound
	}

	if c.Discovery.Password == "" {
		return ErrDiscoveryPassNotFound
	}

	if c.Discovery.SSHPort == "" {
		c.Discovery.SSHPort = "22"
	}

	if c.Discovery.APIPort == "" {
		c.Discovery.APIPort = "8728"
	}

	if c.Discovery.APISSLPort == "" {
		c.Discovery.APISSLPort = "8729"
	}

	return nil
}
