package api

import (
	"fmt"
	"strings"
)

type ValidKeyBody struct {
	Success bool   `json:"success"` // need to capitalise these so json unmarshelling packages can see/use them
	Cause   string `json:"cause"`
}

// CheckApiKey - Validate an API key.
func CheckApiKey(cl *HypixelApiClient) (bool, error) {
	var jsonDecodedBody ValidKeyBody
	// Note: NOT a valid endpoint in Hypixel API v2. But we could use literally anything else and it'd tell us if the API key is invalid or not, so works.
	err := cl.Get(BaseApiUrl+"key", &jsonDecodedBody)
	if err != nil {
		if !strings.Contains(err.Error(), "invalid status") {
			return false, err
		}
	}

	if jsonDecodedBody.Success {
		return false, fmt.Errorf("this should not be successful... what")
	} else {
		// status code would be better but meh
		if strings.Contains(jsonDecodedBody.Cause, "Invalid") {
			return false, fmt.Errorf("invalid api key")
		}
	}

	return true, nil
}
