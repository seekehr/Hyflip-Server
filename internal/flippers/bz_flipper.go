package flippers

import (
	"Hyflip-Server/internal/api"
	"Hyflip-Server/internal/config"
	"fmt"
)

type Response struct {
	Success     bool               `json:"success"`
	LastUpdated int64              `json:"lastUpdated"`
	Products    map[string]Product `json:"products"`
}

type Product struct {
	ProductID   string         `json:"product_id"`
	SellSummary []OrderSummary `json:"sell_summary"`
	BuySummary  []OrderSummary `json:"buy_summary"`
	QuickStatus QuickStatus    `json:"quick_status"`
}

type OrderSummary struct {
	Amount       int     `json:"amount"`
	PricePerUnit float64 `json:"pricePerUnit"`
	Orders       int     `json:"orders"`
}

type QuickStatus struct {
	ProductID      string  `json:"productId"`
	SellPrice      float64 `json:"sellPrice"`
	SellVolume     int     `json:"sellVolume"`
	SellMovingWeek int     `json:"sellMovingWeek"`
	SellOrders     int     `json:"sellOrders"`
	BuyPrice       float64 `json:"buyPrice"`
	BuyVolume      int     `json:"buyVolume"`
	BuyMovingWeek  int     `json:"buyMovingWeek"`
	BuyOrders      int     `json:"buyOrders"`
}

type FoundFlip struct {
	Profit  int    `json:"profit"`
	ItemId  int    `json:"itemId"`
	Command string `json:"command"`
}

const BAZAAR_TAX = 1.25

func Flip(cl *api.HypixelApiClient, config *config.BZConfig) (*FoundFlip, error) {
	var resp Response
	err := cl.Get(api.SbApiUrl+"bazaar", &resp)
	if err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf("bzflip not successful")
	}

	for _, product := range resp.Products {
		profit := int(CalculateWithTax(float32(product.QuickStatus.BuyPrice - product.QuickStatus.SellPrice)))
		if profit < config.MinProfit {
			continue
		}

		profitPercentage := int(float64(profit) / product.QuickStatus.SellPrice * 100)
		if profitPercentage < config.MinProfitPercentage {
			continue
		}

		buyVol := product.QuickStatus.BuyVolume
		sellVol := product.QuickStatus.SellVolume
		if buyVol < config.MinBuyVolume { // low demand
			continue
		}
		if buyVol-sellVol < config.MinVolumeDiff { // should have at least this much diff in demand compared to supply
			continue
		}

		buyMovingWeek := product.QuickStatus.BuyMovingWeek
		sellMovingWeek := product.QuickStatus.SellMovingWeek
		if buyMovingWeek < config.MinBuyMovingWeek || sellMovingWeek < config.MinSellMovingWeek {
			continue
		}
	}
}

// CalculateWithTax calculates the profit you get after bazaar tax cut.
func CalculateWithTax(num float32) float32 {
	return (BAZAAR_TAX / 100) * num
}
