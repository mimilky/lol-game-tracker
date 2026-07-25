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

type rawMatchDetail struct {
	ParticipantData []struct {
		ParticipantId int `json:"participantId"`
		PlayerDetail  struct {
			PUUID string `json:"puuid"`
		} `json:"player"`
	} `json:"participantIdentities"`
	Participants []struct {
		ParticipantId int `json:"participantId"`
		Stats         struct {
			Win bool `json:"win"`
		} `json:"stats"`
		Timeline struct {
			Lane string `json:"lane"`
			Role string `json:"role"`
		} `json:"timeline"`
	} `json:"participants"`
}

func GetMatchHistory(client *http.Client, auth lockfile.LCUAuth, puuid string) ([]int64, error) { //GameIDを返す
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

type Participant struct {
	ParticipantID int
	PUUID         string
	Lane          string
	Role          string
	Win           bool
}

func mergeParticipants(detail rawMatchDetail) []Participant {
	puuidByID := make(map[int]string, len(detail.ParticipantData))
	for _, pi := range detail.ParticipantData {
		puuidByID[pi.ParticipantId] = pi.PlayerDetail.PUUID
	}
	participants := make([]Participant, 0, len(detail.Participants))
	for _, p := range detail.Participants {
		participants = append(participants, Participant{
			ParticipantID: p.ParticipantId,
			PUUID:         puuidByID[p.ParticipantId],
			Lane:          p.Timeline.Lane,
			Role:          p.Timeline.Role,
			Win:           p.Stats.Win,
		})
	}
	return participants
}

func GetMatchDetail(client *http.Client, auth lockfile.LCUAuth, gameID int64) ([]Participant, error) {
	body, err := get(client, auth, fmt.Sprintf("/lol-match-history/v1/games/%d", gameID))
	if err != nil {
		return nil, err
	}
	var detail rawMatchDetail
	if err := json.Unmarshal(body, &detail); err != nil {
		return nil, fmt.Errorf("JSONのパースに失敗: %w", err)
	}
	return mergeParticipants(detail), nil
}
