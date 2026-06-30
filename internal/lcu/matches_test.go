package lcu

import (
	"fmt"
	"testing"

	"lol-game-tracker/internal/lockfile"
)

// TestGetMatchHistoryJSON は GetMatchHistory が LCU から実際に受け取る対戦履歴 JSON を表示する。
// LCU（LeagueClientUx.exe）が起動していることを前提とする。
// 実行: go test ./internal/lcu/ -run TestGetMatchHistoryJSON -v
func TestGetMatchHistoryJSON(t *testing.T) {
	lockfilePath, err := lockfile.Path()
	if err != nil {
		t.Fatalf("lockfile の特定に失敗（LCU は起動しているか）: %v", err)
	}
	auth, err := lockfile.Read(lockfilePath)
	if err != nil {
		t.Fatalf("lockfile の読み込みに失敗: %v", err)
	}

	client := NewClient()
	puuid, err := GetPUUID(client, auth)
	if err != nil {
		t.Fatalf("PUUID の取得に失敗: %v", err)
	}

	// GetMatchHistory が内部で受け取るのと同じ生 JSON を取得して表示
	body, err := get(client, auth, fmt.Sprintf("/lol-match-history/v1/products/lol/%s/matches", puuid))
	if err != nil {
		t.Fatalf("対戦履歴の取得に失敗: %v", err)
	}
	t.Logf("受信 JSON (%d bytes):\n%s", len(body), body)
}

// TestGetMatchDetailJSON は試合詳細エンドポイントから受け取る JSON を表示する。
// getMatchDetail は未実装のため、同じ非公開 get() で /lol-match-history/v1/games/{gameId} を直接叩く。
// gameId は GetMatchHistory（queueId フィルタ済み）の先頭を使う。LCU 起動を前提とする。
// 実行: go test ./internal/lcu/ -run TestGetMatchDetailJSON -v
func TestGetMatchDetailJSON(t *testing.T) {
	lockfilePath, err := lockfile.Path()
	if err != nil {
		t.Fatalf("lockfile の特定に失敗（LCU は起動しているか）: %v", err)
	}
	auth, err := lockfile.Read(lockfilePath)
	if err != nil {
		t.Fatalf("lockfile の読み込みに失敗: %v", err)
	}

	client := NewClient()
	puuid, err := GetPUUID(client, auth)
	if err != nil {
		t.Fatalf("PUUID の取得に失敗: %v", err)
	}

	gameIDs, err := GetMatchHistory(client, auth, puuid)
	if err != nil {
		t.Fatalf("対戦履歴の取得に失敗: %v", err)
	}
	if len(gameIDs) == 0 {
		t.Skip("対象の試合が見つからないため試合詳細テストをスキップ")
	}

	gameID := gameIDs[0]
	body, err := get(client, auth, fmt.Sprintf("/lol-match-history/v1/games/%d", gameID))
	if err != nil {
		t.Fatalf("試合詳細の取得に失敗 (gameId=%d): %v", gameID, err)
	}
	t.Logf("gameId=%d 受信 JSON (%d bytes):\n%s", gameID, len(body), body)
}
