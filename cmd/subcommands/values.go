package cmd

import (
	"github.com/fatih/color"
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
var config *Config
