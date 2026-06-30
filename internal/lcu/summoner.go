package lcu

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"lol-game-tracker/internal/lockfile"
)

func GetPUUID(client *http.Client, auth lockfile.LCUAuth) (string, error) {
	body, err := get(client, auth, "/lol-summoner/v1/current-summoner")
	if err != nil {
		return "", err
	}
	var userData map[string]any
	if err := json.Unmarshal(body, &userData); err != nil {
		return "", fmt.Errorf("JSONのパースに失敗: %w", err)
	}
	puuid, ok := userData["puuid"].(string)
	if !ok {
		return "", errors.New("puuidが見つかりません")
	}
	return puuid, nil
}
