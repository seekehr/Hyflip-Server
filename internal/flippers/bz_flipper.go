package flippers

import (
	"Hyflip-Server/internal/api"
	"Hyflip-Server/internal/cache"
	"Hyflip-Server/internal/config"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

const PriceHistoryTimeSpan = 3600 * 24 * 7 // 1 week

type BazaarResponse struct {
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

type filteredProductInfo struct {
	Profit         int `json:"profit"`
	SellVolume     int `json:"sellVolume"`
	SellMovingWeek int `json:"sellMovingWeek"`
	BuyVolume      int `json:"buyVolume"`
	BuyMovingWeek  int `json:"buyMovingWeek"`
}

type BazaarFoundFlip struct {
	ProductID                       string  `json:"productId"`
	Command                         string  `json:"command"`
	Profit                          int     `json:"profit"`
	SellPrice                       float64 `json:"sellPrice"`
	BuyPrice                        float64 `json:"buyPrice"`
	SellVolume                      int     `json:"sellVolume"`
	SellMovingWeek                  int     `json:"sellMovingWeek"`
	BuyVolume                       int     `json:"buyVolume"`
	BuyMovingWeek                   int     `json:"buyMovingWeek"`
	RecommendedFlipVolume           int     `json:"recommendedFlipVolume"`
	ProfitFromRecommendedFlipVolume int     `json:"profitFromRecommendedFlipVolume"`
}

const (
	VolumeAverageCheck       = 8
	MaxSwingPercentage       = 40
	RecommendedBuyPercentage = 0.01 // 1%;arbitrary for now
	BazaarTax                = 1.25
)

// GetBzFlipsForUser is used to apply a user's config filter upon the flips data received from the cache. USE THIS FUNCTION FOR EVERYTHING. BZFLIP IS USED FOR UPDATING CACHE.
func GetBzFlipsForUser(userConfig *config.BZConfig, bzCache *cache.Cache[<-chan BazaarFoundFlip]) (<-chan BazaarFoundFlip, error) {
	resultsChan := make(chan BazaarFoundFlip, 200)
	go func() { // goroutine is needed not because this function is very costly but because we will be receiving the bzCache slowly (upon update at least; will be fast enough if returned cached) and we js wanna filter n pass it on to the API
		for foundFlip := range bzCache.Get() {
			log.Println("Filtered one product received from cache.")
			filteredProduct := filter(nil, &foundFlip, userConfig)
			if filteredProduct == nil {
				continue
			}
			resultsChan <- foundFlip // we just send our existing flip struct as we know no values will change. filter just filters it and returns info we already knew like profit
		}
	}()

	return resultsChan, nil
}

// BzFlip returns a channel of found flips (for efficiency purposes). It uses the config to filter items and then checks for market manipulation using `price_checker.go`. Used for cache updates.
func BzFlip(cl *api.HypixelApiClient, config *config.BZConfig) (<-chan BazaarFoundFlip, error) {
	reqTime := time.Now()
	var resp BazaarResponse
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
	resultsChan := make(chan BazaarFoundFlip, 200)
	var (
		wg sync.WaitGroup
	)

	// worker pool for market manipulation checker
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

				recomFlipVol := int(float64(product.BuyMovingWeek/VolumeAverageCheck) * RecommendedBuyPercentage)
				profitFromRecom := product.Profit * recomFlipVol
				resultsChan <- BazaarFoundFlip{
					ProductID:                       product.ProductID,
					Command:                         "/bzs " + product.ProductID,
					Profit:                          product.Profit,
					SellPrice:                       product.SellPrice,
					BuyPrice:                        product.BuyPrice,
					SellVolume:                      product.SellVolume,
					SellMovingWeek:                  product.SellMovingWeek,
					BuyVolume:                       product.BuyVolume,
					BuyMovingWeek:                   product.BuyMovingWeek,
					RecommendedFlipVolume:           recomFlipVol,
					ProfitFromRecommendedFlipVolume: profitFromRecom,
				}
			}
		}()
	}

	// Manager goroutine to close the channels
	go func() {
		filteringTime := time.Now()
		for _, product := range resp.Products {
			filteredProduct := filter(&product, nil, config)
			if filteredProduct == nil { // product does not match our given filters
				continue
			}

			// copy everytime but allg ig. if we sent *Product then it would just point to the latest variable in the loop as the variable will be re-used
			// so 0xUWU would replace 0x322 as product after an iteration
			respectableProducts <- api.PriceHistoryProduct{
				ProductID:      product.ProductID,
				Profit:         filteredProduct.Profit,
				SellPrice:      product.QuickStatus.SellPrice,
				BuyPrice:       product.QuickStatus.BuyPrice,
				SellVolume:     filteredProduct.SellVolume,
				SellMovingWeek: filteredProduct.SellMovingWeek,
				BuyVolume:      filteredProduct.BuyVolume,
				BuyMovingWeek:  filteredProduct.BuyMovingWeek,
			}
		}
		close(respectableProducts) // no more work for the price history checking goroutine
		log.Println("Filtering products took time: " + time.Since(filteringTime).String())
		marketManipTime := time.Now()
		wg.Wait()          // wait for price checking to be done so we can confirm all flips
		close(resultsChan) // no more work for the caller of this function. everything DONE
		log.Println("Market manipulation took (ESTIMATED): " + time.Since(marketManipTime).String())
	}()

	return resultsChan, nil
}

// filter using a config and EITHER Product or BazaarFoundFlip.
func filter(product *Product, bzFlip *BazaarFoundFlip, bzConfig *config.BZConfig) *filteredProductInfo {
	var (
		productId      string
		sellPrice      float64
		buyPrice       float64
		sellVolume     int
		buyVolume      int
		sellMovingWeek int
		buyMovingWeek  int
	)

	if product != nil {
		// one hell of a one-liner huh
		productId, sellPrice, buyPrice, sellVolume, buyVolume, sellMovingWeek, buyMovingWeek = product.ProductID, product.QuickStatus.SellPrice, product.QuickStatus.BuyPrice, product.QuickStatus.SellVolume, product.QuickStatus.BuyVolume, product.QuickStatus.SellMovingWeek, product.QuickStatus.BuyMovingWeek
	} else if bzFlip != nil {
		productId, sellPrice, buyPrice, sellVolume, buyVolume, sellMovingWeek, buyMovingWeek = bzFlip.ProductID, bzFlip.SellPrice, bzFlip.BuyPrice, bzFlip.SellVolume, bzFlip.BuyVolume, bzFlip.SellMovingWeek, bzFlip.BuyMovingWeek
	} else {
		return nil // both product and bzFlip cannot be nil
	}

	// Name is excluded. Can be "COBBLE" e.g. to exclude COBBLESTONE, ENCHANTED_COBBLESTONE and so on
	if isIdExcluded(productId, bzConfig) {
		//log.Println("Ignoring product: " + product.ProductID + ". Cause: EXCLUDED_ITEMS.")
		return nil
	}

	taxFactor := 1 - BazaarTax/100.0 // 0.9875 if tax = 1.25%
	profit := (buyPrice - sellPrice) * taxFactor

	if profit < float64(bzConfig.MinProfit) {
		//log.Println("Ignoring product: " + product.ProductID + ". Cause: MIN_PROFIT (" + strconv.FormatFloat(profit, 'f', 2, 64) + ").")
		return nil
	}

	profitPercentage := profit / sellPrice * 100
	if profitPercentage < float64(bzConfig.MinProfitPercentage) {
		//log.Println("Ignoring product: " + product.ProductID + ". Cause: MIN_PROFIT_% (" + strconv.FormatFloat(profitPercentage, 'f', 2, 64) + ").")
		return nil
	}

	if buyVolume < bzConfig.MinBuyVolume { // low demand
		//log.Println("Ignoring product: " + product.ProductID + ". Cause: MIN_BUY_VOL (" + strconv.Itoa(buyVol) + ").")
		return nil
	}
	if buyVolume-sellVolume < bzConfig.MinVolumeDiff { // should have at least this much diff in demand compared to supply
		//log.Println("Ignoring product: " + product.ProductID + ". Cause: MIN_BUY_VOL_DIFF (" + strconv.Itoa(buyVol-sellVol) + ").")
		return nil
	}

	// less moving week buy
	if buyMovingWeek < bzConfig.MinBuyMovingWeek || sellMovingWeek < bzConfig.MinSellMovingWeek {
		//log.Println("Ignoring product: " + product.ProductID + ". Cause: MIN_BUY_/_SELL_WEEK (" + strconv.Itoa(buyMovingWeek) + "/" + strconv.Itoa(sellMovingWeek) + ").")
		return nil
	}

	// to ensure there is enough daily demand that you dont have to do weekly flips lol
	if buyMovingWeek/VolumeAverageCheck <= 10 || sellMovingWeek/VolumeAverageCheck <= 10 {
		//log.Println("Ignoring product: " + product.ProductID + ". Cause: MIN_DAILY_BUY_/_SELL_WEEK (" + strconv.Itoa(buyMovingWeek / VolumeAverageCheck) + "/" + strconv.Itoa(sellMovingWeek / VolumeAverageCheck) + ").")
		return nil
	}

	//log.Println("Respectable product " + product.ProductID + " found.")
	return &filteredProductInfo{
		Profit:         int(profit),
		SellVolume:     sellVolume,
		SellMovingWeek: sellMovingWeek,
		BuyVolume:      buyVolume,
		BuyMovingWeek:  buyMovingWeek,
	}
}

func isIdExcluded(itemId string, config *config.BZConfig) bool {
	for _, v := range config.ExcludeItems {
		if strings.Contains(v, itemId) {
			return true
		}
	}
	return false
}
