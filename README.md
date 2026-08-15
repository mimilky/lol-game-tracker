# lol-game-tracker

## 1. 概要

League of Legends のカスタムゲームデータアップロード用クライアント。

LoLクライアント（LCU）からローカルの対戦履歴を取得し、**カスタムゲームの戦績のみ**を抽出して中央サーバへアップロードを行う。常駐せず、実行するとその時点のデータを送信して終了するワンショット型のツール。

## 2. 環境構築

**Windows 環境**を想定（Go 1.21 以上）。

### 1. ビルド（実行ファイルの生成）

リポジトリ直下で以下を実行し、`lol-game-tracker.exe` を生成。

```sh
go build -ldflags="-H windowsgui" -o lol-game-tracker.exe .
```

- `-ldflags="-H windowsgui"` によりコンソール窓を出さず、結果はポップアップ（成功=「完了」／失敗=エラー内容）で表示される。

### 2. 実行

**League of Legends クライアントを起動した状態**で、生成した `lol-game-tracker.exe` を実行する（ダブルクリック、またはコマンドから）。

```sh
./lol-game-tracker.exe
```

- LoL クライアントが起動していないと、接続情報（lockfile）を取得できず失敗する。
- 実行が完了すると「完了」ポップアップが表示されます。失敗時はエラー内容がポップアップに表示される。
