// Copyright 2021 stafiprotocol
// SPDX-License-Identifier: LGPL-3.0-only

package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type ConfigInit struct {
	EndpointList []string // url for  rpc endpoint
	KeystorePath string

	StakeManagerProgramID string
	MintManagerProgramID  string

	StakeManagerAccount string
	MintManagerAccount  string

	// init related
	RSolMintAddress     string
	FeeRecipientAddress string
	ValidatorAddress    string
	BridgeSignerAddress string

	FeePayerAccount string
	AdminAccount    string

	Bond             uint64
	Unbond           uint64
	Active           uint64
	LatestEra        uint64
	Rate             uint64
	TotalRSolSupply  uint64
	TotalProtocolFee uint64
}

func LoadInitConfig(configFilePath string) (*ConfigInit, error) {
	var cfg = ConfigInit{}
	if err := loadSysConfigInit(configFilePath, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
func loadSysConfigInit(path string, config *ConfigInit) error {
	_, err := os.Open(path)
	if err != nil {
		return err
	}
	if _, err := toml.DecodeFile(path, config); err != nil {
		return err
	}
	fmt.Println("load config success")
	return nil
}

type ConfigSettings struct {
	EndpointList []string // url for  rpc endpoint
	KeystorePath string

	StakeManagerProgramID string
	MintManagerProgramID  string
	StakeManagerAddress   string
	MintManagerAddress    string

	FeePayerAccount string
	AdminAccount    string

	// set related
	RateChangeLimit        uint64
	UnbondingDuration      uint64
	UnstakeFeeCommission   uint64
	AddValidatorAddress    string
	RemoveValidatorAddress string
	NewAdminAddress        string
	NewFeeRecipientAddress string
	MintAuthorities        []string
}

func LoadSettingsConfig(configFilePath string) (*ConfigSettings, error) {
	var cfg = ConfigSettings{}
	if err := loadSysConfigSettings(configFilePath, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func loadSysConfigSettings(path string, config *ConfigSettings) error {
	_, err := os.Open(path)
	if err != nil {
		return err
	}
	if _, err := toml.DecodeFile(path, config); err != nil {
		return err
	}
	fmt.Println("load config success")
	return nil
}

type ConfigStart struct {
	EndpointList []string // url for  rpc endpoint
	LogFilePath  string
	KeystorePath string

	StakeManagerProgramID string
	MintManagerProgramID  string

	StakeManagerAddress string
	MintManagerAddress  string

	FeePayerAccount string
}

func LoadStartConfig(configFilePath string) (*ConfigStart, error) {
	var cfg = ConfigStart{}
	if err := loadSysConfigStart(configFilePath, &cfg); err != nil {
		return nil, err
	}
	if len(cfg.LogFilePath) == 0 {
		cfg.LogFilePath = "./log_data"
	}

	return &cfg, nil
}

func loadSysConfigStart(path string, config *ConfigStart) error {
	_, err := os.Open(path)
	if err != nil {
		return err
	}
	if _, err := toml.DecodeFile(path, config); err != nil {
		return err
	}
	fmt.Println("Load config success")
	return nil
}
