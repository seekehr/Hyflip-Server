package api

import (
	"fmt"
	"github.com/bytedance/sonic"
	"github.com/valyala/fasthttp"
	"log"
	"strconv"
)

// HypixelApiClient - One client per API key. Name is kind of misleading lol, this client is also used for all kind of other reqs.
type HypixelApiClient struct {
	ApiKey string
	Client *fasthttp.Client
}

func Init(apiKey string) *HypixelApiClient {
	client := &fasthttp.Client{
		MaxConnsPerHost: 1000, // LIVE connections per host. 10 is arbitrary for now
	}
	// Per Host also means Per API Key here since we're creating a new client for EVERY Api Key
	return &HypixelApiClient{
		ApiKey: apiKey,
		Client: client,
	}
	// every API Key is... well, one API key since for now we're only using the internal API Key without support for user API Keys so yeah.
}

// Get is used to send requests efficiently (and without boilerplate) to (theoretically) ANY url. It sets the API key header, and if successful with the request, uses `dst` to unmarshall the data so you can handle any further errors yourself.
func (cl *HypixelApiClient) Get(url string, dst any) error {
	// fast http performance thing. request/responses pool to prevent GC usage basically
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)
	defer fasthttp.ReleaseRequest(req)
	req.SetRequestURI(url)
	req.Header.Set("API-Key", cl.ApiKey)
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Keep-Alive", "timeout=30, max=100")

	if err := cl.Client.Do(req, resp); err != nil {
		return err
	}
	if resp.StatusCode() == 500 {
		log.Println("Encountered rate-limit status code " + strconv.Itoa(resp.StatusCode()) + " for url " + url)
	}
	
	if resp.StatusCode() != 200 { // might need to change in future; 200 is a bit too general but works for pricechecker & hypixel api for now.
		return fmt.Errorf("invalid status code; " + resp.String())
	}

	// sonic is MUCH faster. uses SIMD.
	return sonic.Unmarshal(resp.Body(), dst)
}
