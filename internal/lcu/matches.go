package lcu

import (
	"encoding/json"
	"fmt"
	"net/http"

	"lol-game-tracker/internal/lockfile"
)

type gameData struct {
	Games struct {
		Games []struct {
			GameId  int64 `json:"gameId"`
			QueueId int   `json:"queueId"`
		} `json:"games"`
	} `json:"games"`
}

type matchDetail struct {
	ParticipantData []struct {
		PlayerDetail struct {
			PUUID string `json:"puuid"`
		} `json:"player"`
	} `json:"participantIdentities"`
}

func GetMatchHistory(client *http.Client, auth lockfile.LCUAuth, puuid string) ([]int64, error) {
	matchHistory, err := get(client, auth, fmt.Sprintf("/lol-match-history/v1/products/lol/%s/matches", puuid))
	if err != nil {
		return nil, err
	}

	var data gameData
	err = json.Unmarshal(matchHistory, &data)
	if err != nil {
		return nil, fmt.Errorf("JSONのパースに失敗: %w", err)
	}
	ids := make([]int64, 0, len(data.Games.Games))
	for _, g := range data.Games.Games {
		const (
			blindCustom = 3100
			draftCustom = 3130
		)
		if g.QueueId != blindCustom && g.QueueId != draftCustom {
			continue
		}
		ids = append(ids, g.GameId)
	}
	return ids, nil
}

func GetMatchDetail(client *http.Client, auth lockfile.LCUAuth, gameID int64) ([]string, error) {
	body, err := get(client, auth, fmt.Sprintf("/lol-match-history/v1/games/%d", gameID))
	if err != nil {
		return nil, err
	}
	var detail matchDetail
	err = json.Unmarshal(body, &detail)
	participantId := make([]string, 0, len(detail.ParticipantData))
	for _, g := range detail.ParticipantData {
		participantId = append(participantId, g.PlayerDetail.PUUID)
	}
	return participantId, err
}
