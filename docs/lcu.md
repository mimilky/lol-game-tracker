# internal/lcu — LCU ローカル API クライアント

LCU（League Client Update）のローカル API（`https://127.0.0.1:{port}`）へアクセスし、PUUID・対戦履歴・試合詳細を取得するパッケージ。OS 非依存（Windows 依存は `lockfile` に隔離）。

依存: `lcu → lockfile`（共有型 `lockfile.LCUAuth` を引数に取るため）。

## ファイル

| ファイル | 内容 |
|---|---|
| `client.go` | `NewClient()`, 非公開 `get()` |
| `summoner.go` | `GetPUUID()` |
| `matches.go` | `GetMatchHistory()`, `GetMatchDetail()`, `Participant`, 非公開 `mergeParticipants()` |
| `matches_test.go` | 実機統合テスト |

## 関数・型

### `NewClient() *http.Client`
`InsecureSkipVerify: true` ＋ `Timeout: 5s` の HTTP クライアント（LCU の自己署名証明書対策）。**外部サーバへは流用しない**。

### `get(client, auth, path) ([]byte, error)`（非公開）
`https://127.0.0.1:{port}{path}` へ Basic 認証（ユーザー固定 `riot`）付き GET を行い、生レスポンスボディを返す共通ヘルパー。`NewRequest`→`SetBasicAuth`→`Do`→`ReadAll` の重複を集約。**HTTP ステータス未検証**（404 のエラー本文でも空データが混入し得る＝要整備）。

### `GetPUUID(client, auth) (string, error)`
`GET /lol-summoner/v1/current-summoner`。内部で `get()` を呼び、`map[string]any` にデコードして `puuid` を取り出す。

### `GetMatchHistory(client, auth, puuid) ([]int64, error)`
`GET /lol-match-history/v1/products/lol/{puuid}/matches`。非公開構造体 `gameData` で **`games.games[].gameId` と `queueId`** を抽出し、**カスタムゲームのみ**に絞った gameId 一覧を返す。
- 判定は `queueId == 3100`（ブラインドピックカスタム）/ `3130`（ドラフトカスタム）。
- 設計仕様 §4 の `queueId == 0` とは**不一致**。コードが正なら設計仕様側を更新する。
- `gameData` は必要な階層のみを `json` タグ付きでモデル化（レスポンス全体は持たない）。

### `GetMatchDetail(client, auth, gameID int64) ([]Participant, error)`
`GET /lol-match-history/v1/games/{gameId}`（**単一 gameId**）。非公開 `rawMatchDetail` で `participantIdentities[]`（puuid）と `participants[]`（`stats.win`・`timeline.lane`/`role`）をパースし、非公開 `mergeParticipants` が **`participantId` をキーに結合**して参加者一覧（通常10人）を返す。
- スライスを丸ごと渡すと URL が壊れる（過去のバグ）。必ず 1 件ずつ渡す。

### `Participant{ ParticipantID, PUUID, Lane, Role, Win }`
試合詳細の1参加者。
- **`lane`/`role` は Riot の推定値でカスタムゲームでは不正確**（TOP が `JUNGLE`/`NONE` になる等）。信頼できるのは `PUUID` と `Win`。lane/role は JSON の値をそのまま反映しており、結合ミスではない。
- JSON タグ無し＝送信時のキーは PascalCase（`upload` パッケージ・GAS 側と一致必須）。

### `mergeParticipants(detail rawMatchDetail) []Participant`（非公開・純粋関数）
`participantIdentities`（puuid）を `map[int]string` 化し、`participants`（勝敗・lane/role）を `participantId` で突き合わせて結合。O(n) でネストループを避ける。LCU 起動なしで単体テスト可能。

## テスト（matches_test.go）

`TestGetMatchHistoryJSON` / `TestGetMatchDetailJSON` は**実機統合テスト**: 実 LCU から生 JSON を取得して `t.Logf` で表示する。`package lcu` の内部テストで非公開 `get()` を直接呼ぶ。**LCU 未起動だと `t.Fatalf` で失敗**するので、未起動環境で `go test ./...` を回すと落ちる（環境変数ガードは未導入）。

## 既知の負債

- `get()` の HTTP ステータス未検証。
- `GetMatchHistory` のコメント（`//GameIDを返す`）や `NewClient`/`GetPUUID` の doc コメントが Go 慣習（識別子名で始める）に沿っていない。
- `rawMatchDetail` に未使用フィールドが混在し得る。
