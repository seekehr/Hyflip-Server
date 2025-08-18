package config

type Config struct {
	MinProfit           int      `json:"min_profit"`
	MinProfitPercentage int      `json:"min_profit_percentage"`
	ExcludeItems        []string `json:"exclude_items"`
	IncludeCraftCost    bool     `json:"include_craft_cost"`
	MinVolume           int      `json:"min_volume"`
	MaxVolume           int      `json:"max_volume"`
	MinInstaBuys        int      `json:"min_insta_buys"`
	MaxInstaSells       int      `json:"min_insta_sells"`
}
