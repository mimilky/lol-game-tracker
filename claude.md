# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## プロジェクトの状況 (Project status)

**初期スカフォールド段階。** `internal/` 配下のドメイン別パッケージ＋ルートの薄い `main` という構成（旧 `main.go` 集約から分割済み）。モジュール初期化済み（`go.mod`: module `lol-game-tracker`、`go 1.26.4`、依存 `golang.org/x/sys`）。設計仕様（後述のセクション 1〜4）の §2 認証情報取得＋ §3 対戦履歴取得＆ gameId 抽出まで実装済み、フィルタ（§4 カスタムゲーム判定）・送信（§5）・差分マーカー（§1/§6）は未実装。

### パッケージ構成

```
main.go                         // run() / main()。各パッケージを順に呼ぶオーケストレーションのみ
internal/lockfile/lockfile.go   // LCU 接続情報の取得（Windows 専用）
internal/lcu/                   // LCU ローカル API クライアント
    client.go                   // NewClient(), get()（共通リクエストヘルパー・非公開）
    summoner.go                 // GetPUUID()
    matches.go                  // GetMatchHistory()
```

依存方向: `main → lcu, lockfile` ／ `lcu → lockfile`（共有型 `LCUAuth` が lockfile 側にあるため）。循環なし。

### 主な関数・型

- `lockfile.LCUAuth{ Password, Port }` — LCU 接続情報。`lcu` パッケージも引数型として参照。
- `lockfile.Path()` — `golang.org/x/sys/windows` の `CreateToolhelp32Snapshot` で `LeagueClientUx.exe` を走査し、`QueryFullProcessImageName`（非公開 `getProcessImagePath`）で実行ファイルのフルパスを得て、同階層の `lockfile` 絶対パスを返す。**Windows 専用**。
- `lockfile.Read(path)` — lockfile を読み、コロン区切り `LeagueClient:<PID>:<port>:<password>:<protocol>` をパースして `LCUAuth` を返す（**JSON ではない**点に注意）。
- `lcu.NewClient()` — `InsecureSkipVerify: true` ＋ `Timeout: 5s` の `*http.Client`（自己署名証明書対策）。
- `lcu.get(client, auth, path)`（非公開）— `https://127.0.0.1:{port}{path}` へ Basic 認証（ユーザー `riot`）付き GET を行い、生のレスポンスボディ `[]byte` を返す共通ヘルパー。`NewRequest`→`SetBasicAuth`→`Do`→`ReadAll` の重複を集約。
- `lcu.GetPUUID(client, auth)` — `GET /lol-summoner/v1/current-summoner`。内部で `get()` を呼び、`map[string]any` にデコードして `puuid` を取り出す。**HTTP ステータス未検証**。
- `lcu.GetMatchHistory(client, auth, puuid)` — `GET /lol-match-history/v1/products/lol/{puuid}/matches`。内部で `get()` を呼び、非公開構造体 `gameData` で **`games.games[].gameId` と `queueId`**（ルート→`games`オブジェクト→`games`配列→各試合）を抽出。**`queueId == 0`（カスタムゲーム）のみ**に絞った **`[]int64`（gameId 一覧）を返す**。`gameData` は必要な階層・フィールドのみを `json` タグ付きでモデル化（レスポンス全体は持たない）。`gameId > last_sync` の差分判定は未実装。
- `main`/`run()` は動作確認用: `lockfile.Path` → `lockfile.Read` → `lcu.GetPUUID` → `lcu.GetMatchHistory` を繋ぎ、結果を stdout、失敗時はエラーを stderr に出して `os.Exit(1)`。`lcu.NewClient()` は1つ生成して両 API 呼び出しで使い回す。エラー処理は `run() error` に集約し `main` で一括処理。

> 検討中のリファクタ: `client`/`auth` を保持する `lcu.Client` 構造体を導入し、`get`/`GetPUUID`/`GetMatchHistory` をメソッド化すれば、各呼び出しでの `client`/`auth` の手渡しを解消できる（未着手）。

### 規約・注意点

- **エラーメッセージは日本語**。下位エラーを包む場合は `fmt.Errorf("〜に失敗: %w", err)`、包む対象がない場合は `errors.New("〜")`（`%w` 引数なしの `fmt.Errorf` は使わない）。`get()` のように既に文脈付きで返すエラーは**再ラップせず** `return ..., err` でそのまま伝播する。ライブラリ層（`internal/*`）は error を返すだけで、stderr 出力・`os.Exit` は `main` のみが行う。
- Windows 依存はプロセス探索（`lockfile` パッケージ）に隔離されており、`lcu` 層は OS 非依存。
- lockfile のポート/パスワードはクライアント起動ごとに変わる**機微情報**。ログ出力やコミットに残さないこと。
- ビルド成果物 `lol-game-tracker.exe` / `src.exe` がリポジトリ直下に残存。**`.gitignore`（`*.exe` 等）が未作成**（まだ何もコミットされていない）。
- モジュールパスは仮に `lol-game-tracker`。公開するなら `go mod edit -module github.com/<user>/lol-game-tracker` で変更。
- `MatchHistory-format.json` — 対戦履歴レスポンスの実サンプル（約229KB、20試合）。`gameData` のタグ階層の根拠であり、パース処理のテスト用フィクスチャに使える。**機微情報を含み得る**ためコミット可否は要確認。
- 直近の作業: gameId ごとの試合詳細取得 → カスタムゲーム判定でフィルタ（§4）→ 中央サーバへ送信（§5）→ `.last_sync` 差分マーカー（§1/§6）。`GetPUUID` の HTTP ステータス検証も整備する。
- 既知の小さな負債: `matches.go` の `GetMatchHistory` 上のコメントが「生バイト列で返す」のままで実態（`[]int64`）と不一致。

> Note: on this Windows checkout the filesystem is case-insensitive, so `CLAUDE.md` and `claude.md` are the **same file** — this guidance and the design spec coexist here by design.

## Commands

No project-specific scripts. Standard Go tooling (module already initialized):

```sh
go build ./...                         # build
go run .                               # run the one-shot tool (main.go is at repo root)
go test ./...                          # run all tests
go test ./internal/lcu/ -run TestGetMatchHistoryJSON -v   # 単一テスト（-v で受信JSON表示）
go vet ./...                           # static checks
go mod tidy                            # sync deps after adding third-party packages
```

実行・テストともに **LoL クライアント（`LeagueClientUx.exe`）起動中**が前提（`lockfile.Path()` がプロセスを要求する）。Windows 専用（`golang.org/x/sys/windows` 依存）。

`internal/lcu/matches_test.go` の `TestGetMatchHistoryJSON` は**実機統合テスト**: 実 LCU から対戦履歴の生 JSON を取得して `t.Logf` で表示する。LCU 未起動だと `t.Fatalf` で失敗するので、`go test ./...` を未起動環境で回すと落ちる点に注意（環境変数ガードは未導入）。`package lcu` の内部テストで非公開 `get()` を直接呼ぶ。

## 入出力


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
