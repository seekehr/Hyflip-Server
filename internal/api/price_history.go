package api

import (
	"fmt"
	"sort"
	"time"
)

type PricePoint struct {
	Buy  float64 `json:"b"`
	Sell float64 `json:"s"`
}

type PriceHistoryProduct struct {
	ProductID      string  `json:"productId"`
	Profit         int     `json:"profit"`
	SellPrice      float64 `json:"sellPrice"`
	BuyPrice       float64 `json:"buyPrice"`
	SellVolume     int     `json:"sellVolume"`
	SellMovingWeek int     `json:"sellMovingWeek"`
	BuyVolume      int     `json:"buyVolume"`
	BuyMovingWeek  int     `json:"buyMovingWeek"`
}

const VolumeAverageCheck = 8
const MaxSwingPercentage = 40

// GetPriceHistory of a product. timeSpan is in minutes.
func GetPriceHistory(cl *HypixelApiClient, itemId string, timeSpan int) ([]PricePoint, error) {
	// timestamp -> PricePoint
	var history map[string]PricePoint
	err := cl.Get(PriceTrackerUrl+itemId, &history)
	if err != nil {
		return nil, fmt.Errorf("error while loading price history: " + err.Error())
	}
	// Extract keys and sort timestamps chronologically. Ugh
	timestamps := make([]string, 0, len(history))
	for ts := range history {
		timestamps = append(timestamps, ts)
	}
	sort.Strings(timestamps)

	relevantPoints := make([]PricePoint, 0)
	for _, ts := range timestamps {
		timestamp, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			return nil, fmt.Errorf("could not parse pricehistory time")
		}

		relevantPoints = append(relevantPoints, history[ts])
		if time.Since(timestamp) > time.Duration(timeSpan)*time.Minute {
			break
		}
	}

	if len(relevantPoints) == 0 {
		return nil, fmt.Errorf("could not find relevant price points")
	}
	return relevantPoints, nil
}

// IsManipulatedBazaarProduct checks for suspiciously sharp price changes with weak volume.
func IsManipulatedBazaarProduct(cl *HypixelApiClient, p *PriceHistoryProduct, timeSpan int) (bool, error) {
	points, err := GetPriceHistory(cl, p.ProductID, timeSpan)
	if err != nil {
		return false, err
	}

	// Find min and max sell prices in the history of the product
	minimum, maximum := p.SellPrice, p.SellPrice
	for _, pt := range points {
		if pt.Sell < minimum {
			minimum = pt.Sell
		}
		if pt.Sell > maximum {
			maximum = pt.Sell
		}
	}

	priceSwing := (maximum - minimum) / ((maximum + minimum) / 2) * 100
	// Check if volumes are abnormally low compared to daily price according weekly moving averages
	lowSellVolume := p.SellVolume < p.SellMovingWeek/VolumeAverageCheck
	lowBuyVolume := p.BuyVolume < p.BuyMovingWeek/VolumeAverageCheck

	// Large swing (40% is arbitrary atm) + low volume = likely manipulation
	if priceSwing > MaxSwingPercentage && (lowSellVolume || lowBuyVolume) {
		return true, nil
	}
	return false, nil
}
