# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

League of Legends クライアント（LCU）からローカルの対戦履歴を取得し、**カスタムゲームの戦績のみ**を抽出して中央サーバ（Google Apps Script）へ送信する、Go 製の軽量ワンショット CLI。常駐せず、ユーザーが任意に実行して完了後即終了する。設計仕様の詳細は後述の §1〜4。

## アーキテクチャ / パッケージ構成

```
main.go                         // run()/main()。各パッケージを順に呼ぶオーケストレーション＋結果のポップアップ通知
internal/lockfile/lockfile.go   // LCU 接続情報の取得（Windows 専用）
internal/lcu/                   // LCU ローカル API クライアント（PUUID/対戦履歴/試合詳細）
internal/upload/upload.go       // 中央サーバ（GAS）への POST
```

依存方向: `main → lcu, lockfile, upload` ／ `lcu → lockfile` ／ `upload → lcu`。循環なし。

処理の流れ（`run()`）: `lockfile.Path` → `lockfile.Read` → `lcu.GetPUUID` → `lcu.GetMatchHistory`（カスタムのみ）→ 各 gameId で `lcu.GetMatchDetail` → `upload.UploadMatchHistory`。`main` は結果を Windows のメッセージボックス（`user32.dll` の `MessageBoxW`）で通知する（成功=「完了」/失敗=エラー本文）。

**各パッケージの詳細設計は [`docs/`](docs/) を参照:**
- [docs/lcu.md](docs/lcu.md) — LCU API クライアント（関数・型、gameData/rawMatchDetail、queueId フィルタ、Participant、テスト、既知の負債）
- [docs/lockfile.md](docs/lockfile.md) — プロセス探索と lockfile パース（Windows 専用）
- [docs/upload.md](docs/upload.md) — GAS への送信、`matchRecord` DTO、GAS 側の運用契約

## 規約 (Conventions)

- **エラーメッセージは日本語**。下位エラーを包む場合は `fmt.Errorf("〜に失敗: %w", err)`、包む対象がない場合は `errors.New("〜")`（`%w` 引数なしの `fmt.Errorf` は使わない）。既に文脈付きで返るエラーは**再ラップせず** `return ..., err`。
- ライブラリ層（`internal/*`）は error を返すだけ。**stderr 出力・`os.Exit`・ポップアップは `main` のみ**が行う。
- **Windows 依存はプロセス探索（`lockfile`）に隔離**。`lcu`/`upload` 層は OS 非依存。
- lockfile のポート/パスワードは起動ごとに変わる**機微情報**。ログ・コミットに残さない。
- LCU の自己署名証明書用クライアント（`InsecureSkipVerify`）は**外部サーバへ流用しない**。
- 必要なフィールドだけを `json` タグ付き構造体でパースする（レスポンス全体はモデル化しない）。
- `MatchHistory-format.json` / `MatchDetail-format.json` は LCU レスポンスの実サンプル（パーステスト用フィクスチャ）。**実プレイヤー名・puuid を含む機微情報**のため `.gitignore` の `*.json` で追跡対象外（`*.exe` は未登録）。
- モジュールパスは仮に `lol-game-tracker`。公開時は `go mod edit -module github.com/<user>/lol-game-tracker`。

## Commands

Go 標準ツールのみ（モジュール初期化済み: `go 1.26.4`、依存 `golang.org/x/sys`）。

```sh
go build ./...                         # コンパイル確認のみ（exe は出力されない点に注意）
go run .                               # 開発実行（コンソールにログ、ポップアップも出る）
go build -ldflags="-H windowsgui" -o lol-game-tracker.exe .   # 配布用 exe（コンソール窓なし・ポップアップのみ）
go test ./...                          # テスト実行
go test ./internal/lcu/ -run TestGetMatchHistoryJSON -v   # 単一テスト（-v で受信JSON表示）
go vet ./...
go mod tidy
```

- 配布用 exe は **`-ldflags="-H windowsgui" -o` 付きで明示ビルド**（`go build ./...` は exe を生成しない＝忘れると古い exe を実行してポップアップが出ない）。
- 実行・テストともに **LoL クライアント（`LeagueClientUx.exe`）起動中**が前提（`lockfile.Path()` がプロセスを要求）。テストは実機統合テスト（詳細は [docs/lcu.md](docs/lcu.md)）。
- Windows 専用（`golang.org/x/sys/windows` 依存）。

---

# 設計仕様

## 1. 概要 (Overview)
本アプリケーションは、League of Legendsのクライアント（LCU: League Client Update）からローカルの対戦履歴データを取得し、カスタムゲーム（インハウス/スクリム）の戦績のみを抽出して中央サーバーへ送信する、Go言語製の軽量クライアントツールです。

常駐型（バックグラウンドプロセス）ではなく、**ユーザーが任意のタイミングで実行し、処理完了後に即座に終了するワンショット（バッチ）形式**を採用しています。

## 2. 技術スタック (Tech Stack)
利用者のPC環境を汚さず、最速・最軽量で動作させるため、巨大なフレームワークは使用せずGoの標準ライブラリを主軸に構築します。

* **言語:** Go (1.21以上を推奨)
* **主要パッケージ (標準ライブラリ):**
  * `net/http` : LCU APIへのGETリクエスト、および中央サーバーへのPOST送信。
  * `crypto/tls` : LCUの自己署名証明書（HTTPSエラー）のスキップ処理 (`InsecureSkipVerify: true`)。
  * `encoding/json` : 取得した履歴データ（JSON）の構造体（Struct）へのパース。
  * `os` / `path/filepath` : `lockfile` やローカル設定ファイル（差分チェック用）の読み書き。
* **サードパーティライブラリ (検討候補):**
  * `github.com/ImOlli/go-lcu` : LoLのインストールパスおよび `lockfile` をWindowsプロセスから自動検出するためのヘルパー。
  * `github.com/go-resty/resty/v2` : (任意) HTTPリクエストの記述を簡略化したい場合。

## 3. 処理フロー (Workflow)
実行ファイル（`.exe`）が起動されてから終了するまでの直列フローです。

1. **差分マーカーの読み込み**
   * ローカルの `.last_sync` ファイルを読み込み、前回送信した最新の試合ID（またはタイムスタンプ）を取得する。
2. **LCU認証情報の取得**
   * LoLクライアントのプロセスから `lockfile` を特定し、「ポート番号」と「Basic認証パスワード」を取得する。
3. **対戦履歴の取得 (GET)**
   * `https://127.0.0.1:{port}/lol-match-history/v1/products/lol/current-summoner/matches` へリクエストを送信し、直近の対戦履歴（JSON）を取得する。
4. **データのフィルタリング**
   * 取得した配列の中から、**カスタムゲーム (`queueId == 0`)** かつ **前回同期以降の新しい試合 (`gameId > last_sync_id`)** のみを抽出する。
   * ※実装は `queueId == 3100`/`3130` を採用（[docs/lcu.md](docs/lcu.md) 参照）。値の整合は要確認。
5. **中央サーバーへの送信 (POST)**
   * 抽出した新規カスタムゲームの配列を、TypeScriptサーバー（APIエンドポイント）へPOST送信する。
6. **差分マーカーの更新と終了**
   * POSTが成功した場合、送信した最新の試合IDで `.last_sync` を上書き保存する。
   * 「送信完了」のメッセージをコンソールに出力し、自動的にプロセスを終了する。

## 4. データモデル (Data Structures)
Go側で定義する主要な構造体（Struct）の設計方針です。LCUからのレスポンス全てをパースする必要はなく、必要なフィールドのみを定義します。

```go
// サーバーへ送信する試合データの構造
type CustomMatch struct {
    GameId       int64  `json:"gameId"`
    GameCreation int64  `json:"gameCreation"`
    QueueId      int    `json:"queueId"` // カスタムゲーム判定用 (0)
    // 参加者情報やスタッツなど必要な項目を追加
}
```
