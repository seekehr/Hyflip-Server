package flippers

import (
	"Hyflip-Server/internal/api"
	"Hyflip-Server/internal/config"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

const PriceHistoryTimeSpan = 3600 * 24 * 7 // 1 week

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
	Profit    int     `json:"profit"`
	ItemId    string  `json:"itemId"`
	Command   string  `json:"command"`
	SellPrice float64 `json:"sellPrice"`
	BuyPrice  float64 `json:"buyPrice"`
}

const BazaarTax = 1.25

// Flip Todo: Cache.
// Flip returns a channel of found flips (for efficiency purposes). It uses the config to filter items and then checks for market manipulation using `price_checker.go`.
func Flip(cl *api.HypixelApiClient, config *config.BZConfig) (<-chan FoundFlip, error) {
	reqTime := time.Now()
	var resp Response
	err := cl.Get(api.SbApiUrl+"bazaar", &resp)
	if err != nil {
		return nil, fmt.Errorf("error while loading bazaar: " + err.Error())
	}
	if !resp.Success {
		return nil, fmt.Errorf("bzflip not successful")
	}
	log.Printf("\nBazaar response success was: %t. Products found: %d. Time taken: %s\n", resp.Success, len(resp.Products), time.Since(reqTime).String())

	// products which pass our initial check, and will now be checked for market manipulating.
	respectableProducts := make(chan api.PriceHistoryProduct, 150)
	// flips
	resultsChan := make(chan FoundFlip, 150)
	var (
		wg sync.WaitGroup
	)

	maxWorkers := 20
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		// Price Checker/Market Manipulation Checker
		go func() {
			defer wg.Done()
			for product := range respectableProducts {
				fr, err := api.IsManipulatedBazaarProduct(cl, &product, PriceHistoryTimeSpan)
				if err != nil {
					continue
				}
				if fr {
					//log.Println(product.ProductID + " is suspected to be market manipulated.")
					continue
				}

				resultsChan <- FoundFlip{
					ItemId:    product.ProductID,
					Profit:    product.Profit,
					Command:   "/bzs " + product.ProductID,
					SellPrice: product.SellPrice,
					BuyPrice:  product.BuyPrice,
				}
			}
		}()
	}

	// Manager goroutine to close the channels
	go func() {
		for _, product := range resp.Products {
			// Name is excluded. Can be "COBBLE" e.g. to exclude COBBLESTONE, ENCHANTED_COBBLESTONE and so on
			if isIdExcluded(product.ProductID, config) {
				//log.Println("Ignoring product: " + product.ProductID + ". Cause: EXCLUDED_ITEMS.")
				continue
			}

			taxFactor := 1 - BazaarTax/100.0 // 0.9875 if tax = 1.25%
			profit := (product.QuickStatus.BuyPrice - product.QuickStatus.SellPrice) * taxFactor

			if profit < float64(config.MinProfit) {
				//log.Println("Ignoring product: " + product.ProductID + ". Cause: MIN_PROFIT (" + strconv.FormatFloat(profit, 'f', 2, 64) + ").")
				continue
			}

			profitPercentage := profit / product.QuickStatus.SellPrice * 100
			if profitPercentage < float64(config.MinProfitPercentage) {
				//log.Println("Ignoring product: " + product.ProductID + ". Cause: MIN_PROFIT_% (" + strconv.FormatFloat(profitPercentage, 'f', 2, 64) + ").")
				continue
			}

			buyVol := product.QuickStatus.BuyVolume
			sellVol := product.QuickStatus.SellVolume
			if buyVol < config.MinBuyVolume { // low demand
				//log.Println("Ignoring product: " + product.ProductID + ". Cause: MIN_BUY_VOL (" + strconv.Itoa(buyVol) + ").")
				continue
			}
			if buyVol-sellVol < config.MinVolumeDiff { // should have at least this much diff in demand compared to supply
				//log.Println("Ignoring product: " + product.ProductID + ". Cause: MIN_BUY_VOL_DIFF (" + strconv.Itoa(buyVol-sellVol) + ").")
				continue
			}

			buyMovingWeek := product.QuickStatus.BuyMovingWeek
			sellMovingWeek := product.QuickStatus.SellMovingWeek
			if buyMovingWeek < config.MinBuyMovingWeek || sellMovingWeek < config.MinSellMovingWeek {
				//log.Println("Ignoring product: " + product.ProductID + ". Cause: MIN_BUY_/_SELL_WEEK (" + strconv.Itoa(buyMovingWeek) + "/" + strconv.Itoa(sellMovingWeek) + ").")
				continue
			}

			//log.Println("Respectable product " + product.ProductID + " found.")

			// copy everytime but allg ig. if we sent *Product then it would just point to the latest variable in the loop as the variable will be re-used
			// so 0xUWU would replace 0x322 as product after an iteration
			respectableProducts <- api.PriceHistoryProduct{
				ProductID:      product.ProductID,
				Profit:         int(profit),
				SellPrice:      product.QuickStatus.SellPrice,
				BuyPrice:       product.QuickStatus.BuyPrice,
				SellVolume:     sellVol,
				SellMovingWeek: sellMovingWeek,
				BuyVolume:      buyVol,
				BuyMovingWeek:  buyMovingWeek,
			}
		}
		close(respectableProducts) // no more work for the price history checking goroutine
		wg.Wait()                  // wait for price checking to be done so we can confirm all flips
		close(resultsChan)         // no more work for the caller of this function
	}()

	return resultsChan, nil
}

func isIdExcluded(itemId string, config *config.BZConfig) bool {
	for _, v := range config.ExcludeItems {
		if strings.Contains(v, itemId) {
			return true
		}
	}
	return false
}
