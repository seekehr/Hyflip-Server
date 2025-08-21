package flippers

import (
	"Hyflip-Server/internal/api"
	"Hyflip-Server/internal/config"
	"fmt"
	"strconv"
	"strings"
	"sync"
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

func Flip(cl *api.HypixelApiClient, config *config.BZConfig) ([]FoundFlip, error) {
	var resp Response
	err := cl.Get(api.SbApiUrl+"bazaar", &resp)
	if err != nil {
		return nil, fmt.Errorf("error while loading bazaar: " + err.Error())
	}
	if !resp.Success {
		return nil, fmt.Errorf("bzflip not successful")
	}
	fmt.Printf("\nBazaar response success was: %t. Products found: %d\n", resp.Success, len(resp.Products))

	// products which pass our initial check, and will now be checked for market manipulating.
	respectableProducts := make(chan api.PriceHistoryProduct, 100)
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		results []FoundFlip
	)

	maxWorkers := 10
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for product := range respectableProducts {
				fr, err := api.IsManipulatedBazaarProduct(cl, &product, PriceHistoryTimeSpan)
				if err != nil {
					continue
				}
				if fr {
					fmt.Println(product.ProductID + " is suspected to be market manipulated.")
					continue
				}

				mu.Lock()
				results = append(results, FoundFlip{
					ItemId:    product.ProductID,
					Profit:    product.Profit,
					Command:   "/bzs " + product.ProductID,
					SellPrice: product.SellPrice,
					BuyPrice:  product.BuyPrice,
				})
				mu.Unlock()
			}
		}()
	}

	fmt.Println("Iterating through found products...")
	for _, product := range resp.Products {
		// Name is excluded. Can be "COBBLE" e.g. to exclude COBBLESTONE, ENCHANTED_COBBLESTONE and so on
		if isIdExcluded(product.ProductID, config) {
			fmt.Println("Ignoring product: " + product.ProductID + ". Cause: EXCLUDED_ITEMS.")
			continue
		}

		profit := int((float64(product.QuickStatus.BuyPrice) - float64(product.QuickStatus.SellPrice)) * BazaarTax / 100.0)
		if profit < config.MinProfit {
			fmt.Println("Ignoring product: " + product.ProductID + ". Cause: MIN_PROFIT (" + strconv.Itoa(profit) + ").")
			continue
		}

		profitPercentage := int(float64(profit) / product.QuickStatus.SellPrice * 100)
		if profitPercentage < config.MinProfitPercentage {
			fmt.Println("Ignoring product: " + product.ProductID + ". Cause: MIN_PROFIT_% (" + strconv.Itoa(profitPercentage) + ").")
			continue
		}

		buyVol := product.QuickStatus.BuyVolume
		sellVol := product.QuickStatus.SellVolume
		if buyVol < config.MinBuyVolume { // low demand
			fmt.Println("Ignoring product: " + product.ProductID + ". Cause: MIN_BUY_VOL (" + strconv.Itoa(buyVol) + ").")
			continue
		}
		if buyVol-sellVol < config.MinVolumeDiff { // should have at least this much diff in demand compared to supply
			fmt.Println("Ignoring product: " + product.ProductID + ". Cause: MIN_BUY_VOL_DIFF (" + strconv.Itoa(buyVol-sellVol) + ").")
			continue
		}

		buyMovingWeek := product.QuickStatus.BuyMovingWeek
		sellMovingWeek := product.QuickStatus.SellMovingWeek
		if buyMovingWeek < config.MinBuyMovingWeek || sellMovingWeek < config.MinSellMovingWeek {
			fmt.Println("Ignoring product: " + product.ProductID + ". Cause: MIN_BUY_/_SELL_WEEK (" + strconv.Itoa(buyMovingWeek) + "/" + strconv.Itoa(sellMovingWeek) + ").")
			continue
		}

		fmt.Println("Respectable product " + product.ProductID + " found.")
		// copy everytime but allg ig. if we sent *Product then it would just point to the latest variable in the loop as the variable will be re-used
		// so 0xUWU would replace 0x322 as product after an iteration
		respectableProducts <- api.PriceHistoryProduct{
			ProductID:      product.ProductID,
			Profit:         profit,
			SellPrice:      product.QuickStatus.SellPrice,
			BuyPrice:       product.QuickStatus.BuyPrice,
			SellVolume:     sellVol,
			SellMovingWeek: sellMovingWeek,
			BuyVolume:      buyVol,
			BuyMovingWeek:  buyMovingWeek,
		}
	}
	close(respectableProducts)
	wg.Wait()
	if len(results) == 0 {
		return nil, fmt.Errorf("no flipped products found")
	}
	return results, nil
}

func isIdExcluded(itemId string, config *config.BZConfig) bool {
	for _, v := range config.ExcludeItems {
		if strings.Contains(v, itemId) {
			return true
		}
	}
	return false
}

// CalculateWithTax calculates the profit you get after bazaar tax cut.
func CalculateWithTax(num float32) float32 {
	return (BazaarTax / 100) * num
}
