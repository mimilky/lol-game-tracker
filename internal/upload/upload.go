package upload

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"lol-game-tracker/internal/lcu"
)

// endpoint は試合データの送信先（Google Apps Script のウェブアプリ）。
const endpoint = "https://script.google.com/macros/s/AKfycbxwyLkTxByLrwjwUZQTtZLZIsv_C8LC5UV4HhF9ql-liXGMXbIdm_0jULk2mhq61gc8GA/exec"

// matchRecord は送信用に gameID を付与した1参加者のレコード。
// lcu.Participant を埋め込むことで GameID・ParticipantID・PUUID・Lane・Role・Win を平坦に出力する。
type matchRecord struct {
	GameID int64
	lcu.Participant
}

// UploadMatchHistory は gameID を付与した participantData を JSON 化して中央サーバへ POST する。
func UploadMatchHistory(client *http.Client, gameID int64, participantData []lcu.Participant) error {
	records := make([]matchRecord, 0, len(participantData))
	for _, p := range participantData {
		records = append(records, matchRecord{GameID: gameID, Participant: p})
	}

	jsonData, err := json.Marshal(records)
	if err != nil {
		return fmt.Errorf("JSONの作成に失敗: %w", err)
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("HTTPリクエストの作成に失敗: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("POSTに失敗: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("サーバがエラーを返しました: %s", resp.Status)
	}
	return nil
}
