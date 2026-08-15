# internal/upload — 中央サーバ（Google Apps Script）への送信（§5）

取得・結合した参加者データを JSON 化し、Google Apps Script のウェブアプリ（`/exec`）へ POST するパッケージ。依存: `upload → lcu`（送信する `lcu.Participant` を参照）。

## 関数・型

### `UploadMatchHistory(client *http.Client, gameID int64, participantData []lcu.Participant) error`
各参加者に `gameID` を付与した非公開 DTO `matchRecord` の配列を JSON 化し、`const endpoint`（GAS `/exec`）へ POST。非 200 はエラー。
- **外部サーバなので `lcu.NewClient()`（`InsecureSkipVerify`）を流用しない**。呼び出し側（`main`）が通常の TLS 検証付き `http.Client{Timeout:...}` を渡す。

### `matchRecord`（非公開 DTO）
```go
type matchRecord struct {
    GameID int64
    lcu.Participant   // 埋め込みで GameID/ParticipantID/PUUID/Lane/Role/Win を平坦に出力
}
```
- **JSON タグ無し＝キーは PascalCase**（`GameID`/`ParticipantID`/`PUUID`/`Lane`/`Role`/`Win`）。GAS 側の読み取りキーと**完全一致必須**（JS は大文字小文字を区別）。

### `const endpoint`
送信先 GAS ウェブアプリの `/exec` URL。現状ハードコード（環境変数化は未実施）。

## GAS 側（サーバ）の運用メモ

`doPost(e)` は Go から届く **JSON 配列**（1試合＝10人分）を受け取り、スプレッドシートへ書き込む。Go リポジトリ外だが、契約として以下を守る。

- **公開範囲は「全員（Anyone）」必須**。未認証の Go クライアントが叩くため、「Google アカウントが必要」だと **401 Unauthorized**（Google が doPost 到達前に拒否）。
- 受信データは**配列**なので `data.forEach(...)` でループする（単一オブジェクト扱いは不可）。
- 読み取りキーは Go の **PascalCase**（`p.GameID` 等）。
- **`gameId` で重複排除**: 既存の GameID はスキップ。判定はバッチ処理前のシート状態を基準にし、ループ中に集合を更新しない（同一試合の残り参加者が誤スキップされるのを防ぐ）。
- **書き込みは `appendRow` ではなく `setValues`（上詰め）**: `appendRow` は下方の残骸を最終行と見なし中間に空白を作るため、基準列（A列=日時）で実データ最終行を求めてその直後へ一括書き込む。
- 反映には**デプロイの更新（新バージョン）**が必要。

## TODO

- `endpoint` の環境変数 / フラグ化。
- Go 側の `.last_sync` 差分マーカー（§1/§6）で再送自体を抑制（現状は毎回送り GAS 側でスキップ）。
