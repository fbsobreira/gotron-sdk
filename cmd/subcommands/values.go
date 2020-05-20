package cmd

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/fatih/color"
	"gopkg.in/yaml.v2"
)

const (
	tronctlDocsDir  = "tronctl-docs"
	defaultNodeAddr = "grpc.trongrid.io:50051"
	defaultTimeout  = 20
)

var (
	g = color.New(color.FgGreen).SprintFunc()
)

// Directories
var (
	// ConfigDir is the directory to store config file
	ConfigDir string
	// DefaultConfigFile is the default config file name
	DefaultConfigFile string
)

// Error strings
var (
	// ErrConfigNotMatch indicates error for no config matchs
	ErrConfigNotMatch = "no config matchs"
	// ErrEmptyEndpoint indicates error for empty endpoint
	ErrEmptyEndpoint = "no endpoint has been set"
)

// Config defines the config schema
type Config struct {
	Node     string `yaml:"node"`
	Ledger   bool   `yaml:"ledger"`
	Verbose  bool   `yaml:"verbose"`
	Timeout  uint32 `yaml:"timeout"`
	NoPretty bool   `yaml:"noPretty"`
}

// ReadConfig represents the current config read from local
var ReadConfig Config

func initConfig() {
	ConfigDir = os.Getenv("HOME") + "/.config/tronctl"
	if err := os.MkdirAll(ConfigDir, 0700); err != nil {
		panic(err.Error())
	}
	DefaultConfigFile = ConfigDir + "/config.default"
	var err error
	ReadConfig, err = LoadConfig()
	if err != nil || ReadConfig.Node == "" {
		if !os.IsNotExist(err) || ReadConfig.Node == "" {
			ReadConfig.Node = defaultNodeAddr
			ReadConfig.Ledger = false
			ReadConfig.Verbose = false
			ReadConfig.Timeout = defaultTimeout
			ReadConfig.NoPretty = false
			out, err := yaml.Marshal(&ReadConfig)
			if err != nil {
				panic(err.Error())
			}
			if err := ioutil.WriteFile(DefaultConfigFile, out, 0600); err != nil {
				panic(fmt.Sprintf("Failed to write to config file %s.", DefaultConfigFile))
			}
		} else {
			panic(err.Error())
		}
	}
}

// LoadConfig loads config file in yaml format
func LoadConfig() (Config, error) {
	in, err := ioutil.ReadFile(DefaultConfigFile)
	if err == nil {
		if err := yaml.Unmarshal(in, &ReadConfig); err != nil {
			return ReadConfig, err
		}
	}
	return ReadConfig, err
}
