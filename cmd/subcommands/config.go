package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func init() {
	cmdConfig := &cobra.Command{
		Use:   "config",
		Short: "update default config",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Help()
			return nil
		},
	}

	cmdConfig.AddCommand([]*cobra.Command{{
		Use:   "set [param] [value]",
		Short: "set default config",
		RunE: func(cmd *cobra.Command, args []string) error {

			var err error
			switch args[0] {
			case "node":
				switch URLcomponents := strings.Split(args[1], ":"); len(URLcomponents) {
				case 1:
					args[1] = args[1] + ":50051"
				}
				config.Node = args[1]
			case "ledger":
				if config.Ledger, err = strconv.ParseBool(args[1]); err != nil {
					return err
				}
			case "verbose":
				if config.Verbose, err = strconv.ParseBool(args[1]); err != nil {
					return err
				}
			case "nopretty":
				if config.NoPretty, err = strconv.ParseBool(args[1]); err != nil {
					return err
				}
			case "apiKey":
				config.APIKey = args[1]
			case "withTLS":
				if config.WithTLS, err = strconv.ParseBool(args[1]); err != nil {
					return err
				}
			default:
				return fmt.Errorf("parameter not found")
			}
			// save to config
			return SaveConfig(config)
		},
	}, {
		Use:   "get [param]",
		Short: "get default config",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "all":
				asJSON, _ := json.Marshal(config)
				fmt.Println(common.JSONPrettyFormat(string(asJSON)))
			case "node":
				fmt.Println(config.Node)
			case "ledger":
				fmt.Println(config.Ledger)
			case "verbose":
				fmt.Println(config.Verbose)
			case "nopretty":
				fmt.Println(config.NoPretty)
			case "apiKey":
				fmt.Println(config.APIKey)
			case "withTLS":
				fmt.Println(config.WithTLS)
			default:
				return fmt.Errorf("parameter not found")
			}
			return nil
		},
	}}...)

	RootCmd.AddCommand(cmdConfig)
}

func initConfig() {
	ConfigDir = os.Getenv("HOME") + "/.config/tronctl"
	if err := os.MkdirAll(ConfigDir, 0700); err != nil {
		panic(err.Error())
	}
	DefaultConfigFile = ConfigDir + "/config.default"
	var err error
	config, err = LoadConfig()
	if err != nil || config.Node == "" {
		if !os.IsNotExist(err) || config.Node == "" {
			config.Node = defaultNodeAddr
			config.Ledger = false
			config.Verbose = false
			config.Timeout = defaultTimeout
			config.NoPretty = false
			config.APIKey = ""
			config.WithTLS = false
			SaveConfig(config)
		} else {
			panic(err.Error())
		}
	}
}

// LoadConfig loads config file in yaml format
func LoadConfig() (*Config, error) {
	in, err := ioutil.ReadFile(DefaultConfigFile)
	readConfig := &Config{}
	if err == nil {
		if err := yaml.Unmarshal(in, readConfig); err != nil {
			return readConfig, err
		}
	}
	return readConfig, err
}

// SaveConfig to yaml
func SaveConfig(conf *Config) error {
	out, err := yaml.Marshal(conf)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(DefaultConfigFile, out, 0600); err != nil {
		panic(fmt.Sprintf("Failed to write to config file %s.", DefaultConfigFile))
	}
	return nil
}
