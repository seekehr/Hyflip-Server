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
	MinVolumeDiff       int      `json:"min_volume_diff"`
	MinBuyVolume        int      `json:"min_buy_volume"`
	MinSellMovingWeek   int      `json:"sell_moving_week"`
	MinBuyMovingWeek    int      `json:"buy_moving_week"`
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
	return &BZConfig{ // very lenient as this is also used for caching so we need as many flips as possible
		ConfigVersion:       "1.0.0",
		MinProfit:           50,
		MinProfitPercentage: 10,
		ExcludeItems:        nil,
		IncludeCraftCost:    false, // TODO
		MinBuyVolume:        5,
		MinVolumeDiff:       10,
		MinSellMovingWeek:   30,
		MinBuyMovingWeek:    30,
	}
}

func (a *AHConfig) Scan(src interface{}) error {
	var b []byte

	switch v := src.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("AHConfig: expected []byte or string, got %T", src)
	}

	return json.Unmarshal(b, a)
}

func (b *BZConfig) Scan(src interface{}) error {
	var data []byte

	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("BZConfig: expected []byte or string, got %T", src)
	}

	return json.Unmarshal(data, b)
}
