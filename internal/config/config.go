package config

import (
	"encoding/json"
	"fmt"
)

type UserConfig struct {
	AhConfig AHConfig
	BzConfig BZConfig
}

type AHConfig struct {
	ConfigVersion       string   `json:"config_version"`
	MinProfit           int      `json:"min_profit"`
	MinProfitPercentage int      `json:"min_profit_percentage"`
	ExcludeItems        []string `json:"exclude_items"`
	IncludeCraftCost    bool     `json:"include_craft_cost"`
	MinVolume           int      `json:"min_volume"`
	MaxVolume           int      `json:"max_volume"`
}

type BZConfig struct {
	ConfigVersion       string   `json:"config_version"`
	MinProfit           int      `json:"min_profit"`
	MinProfitPercentage int      `json:"min_profit_percentage"`
	ExcludeItems        []string `json:"exclude_items"`
	IncludeCraftCost    bool     `json:"include_craft_cost"`
	MinVolume           int      `json:"min_volume"`
	MaxVolume           int      `json:"max_volume"`
	MinInstaBuys        int      `json:"min_insta_buys"`
	MaxInstaSells       int      `json:"min_insta_sells"`
}

func GenerateDefaultAHConfig() *AHConfig {
	return &AHConfig{
		ConfigVersion:       "",
		MinProfit:           1000000,
		MinProfitPercentage: 20,
		ExcludeItems:        nil,
		IncludeCraftCost:    false,
		MinVolume:           20,
		MaxVolume:           20,
	}
}

func GenerateDefaultBZConfig() *BZConfig {
	return &BZConfig{
		ConfigVersion:       "1.0.0",
		MinProfit:           1000,
		MinProfitPercentage: 20,
		ExcludeItems:        nil,
		IncludeCraftCost:    false,
		MinVolume:           1000,
		MaxVolume:           1000,
	}
}

// Scan to allow unmarshalling AHConfig from jsonb.
func (a *AHConfig) Scan(src interface{}) error {
	b, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("AHConfig: expected []byte, got %T", src)
	}
	return json.Unmarshal(b, a)
}

// Scan to allow unmarshalling BZConfig from jsonb.
func (a *BZConfig) Scan(src interface{}) error {
	b, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("AHConfig: expected []byte, got %T", src)
	}
	return json.Unmarshal(b, a)
}
