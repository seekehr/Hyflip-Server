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
	MinProfit           int      `json:"min_profit"`
	MinProfitPercentage int      `json:"min_profit_percentage"`
	ExcludeItems        []string `json:"exclude_items"`
	IncludeCraftCost    bool     `json:"include_craft_cost"`
	MinVolume           int      `json:"min_volume"`
	MaxVolume           int      `json:"max_volume"`
}

type BZConfig struct {
	MinProfit           int      `json:"min_profit"`
	MinProfitPercentage int      `json:"min_profit_percentage"`
	ExcludeItems        []string `json:"exclude_items"`
	IncludeCraftCost    bool     `json:"include_craft_cost"`
	MinVolume           int      `json:"min_volume"`
	MaxVolume           int      `json:"max_volume"`
	MinInstaBuys        int      `json:"min_insta_buys"`
	MaxInstaSells       int      `json:"min_insta_sells"`
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
