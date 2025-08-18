package api

import (
	"errors"
)

type mojangRequest struct {
	Id       string `json:"id"`
	ErrorMsg string `json:"errorMsg"`
}

// GetMojangUuid returns a player uuid from the username. Funny that we're using `HypixelApiClient` for it lol
func GetMojangUuid(cl *HypixelApiClient, username string) (string, error) {
	var id mojangRequest
	if err := cl.Get(MojangUuidApi+username, &id); err != nil {
		return "", errors.New("failed to get mojang uuid") // might be some other error but who cares. we gotta do this because cl already json marshals to `dst` and i dont wanna bother rewriting the entire code without cl
	}

	if id.Id == "" {
		return "", errors.New("no id found")
	}

	return id.Id, nil
}
