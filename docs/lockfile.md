# internal/lockfile — LCU 接続情報の取得（Windows 専用）

実行中の `LeagueClientUx.exe` プロセスから `lockfile` を特定し、その内容から LCU 接続情報（ポート・Basic 認証パスワード）を取り出すパッケージ。**Windows 専用**（`golang.org/x/sys/windows` 依存）。アプリの OS 依存はこのパッケージに隔離されており、`lcu` 層は OS 非依存。

## 関数・型

### `LCUAuth{ Password, Port }`
LCU 接続情報。`lcu` パッケージも引数型として参照する共有型。
- **機微情報**: ポート/パスワードはクライアント起動ごとに変わる。ログ出力やコミットに残さないこと。

### `Path() (string, error)`
`golang.org/x/sys/windows` の `CreateToolhelp32Snapshot` でプロセスを走査し、`LeagueClientUx.exe` を発見。`QueryFullProcessImageName`（非公開 `getProcessImagePath`）で実行ファイルのフルパスを得て、**同階層の `lockfile` の絶対パス**を返す。プロセスが見つからない場合はエラー。

### `getProcessImagePath(pid) (string, error)`（非公開）
指定 PID の実行ファイルのフルパスを取得するヘルパー。

### `Read(path) (LCUAuth, error)`
lockfile を読み、**コロン区切り** `LeagueClient:<PID>:<port>:<password>:<protocol>` をパースして `LCUAuth` を返す。
- **JSON ではない**。`strings.Split(":")` でパースし、`port`=3番目、`password`=4番目のフィールドを使う。
