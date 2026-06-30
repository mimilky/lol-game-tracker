// Package lockfile は実行中の LeagueClientUx プロセスから lockfile を特定し、
// その内容から LCU 接続情報（LCUAuth）を取り出す。Windows 専用。
package lockfile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// LCUAuth は LCU への接続に必要なポートと Basic 認証パスワードを保持する。
type LCUAuth struct {
	Password string
	Port     string
}

// Path は実行中の LeagueClientUx.exe プロセスを探索し、その実行ファイルと
// 同階層に存在する "lockfile" の絶対パスを返す。プロセスが見つからない場合はエラーを返す。
func Path() (string, error) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return "", fmt.Errorf("プロセススナップショットの作成に失敗: %w", err)
	}
	defer windows.CloseHandle(snapshot)

	const leagueClientUxProcessName = "LeagueClientUx.exe"
	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))

	for err = windows.Process32First(snapshot, &entry); err == nil; err = windows.Process32Next(snapshot, &entry) {
		name := windows.UTF16ToString(entry.ExeFile[:])
		if !strings.EqualFold(name, leagueClientUxProcessName) {
			continue
		}

		exePath, err := getProcessImagePath(entry.ProcessID)
		if err != nil {
			return "", err
		}
		return filepath.Join(filepath.Dir(exePath), "lockfile"), nil
	}

	return "", fmt.Errorf("プロセス %q が見つかりません", leagueClientUxProcessName)
}

// getProcessImagePath は指定 PID のプロセスの実行ファイルのフルパスを取得する。
func getProcessImagePath(pid uint32) (string, error) {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return "", fmt.Errorf("プロセス %d のオープンに失敗: %w", pid, err)
	}
	defer windows.CloseHandle(handle)

	buf := make([]uint16, windows.MAX_PATH)
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(handle, 0, &buf[0], &size); err != nil {
		return "", fmt.Errorf("プロセス %d の実行ファイルパス取得に失敗: %w", pid, err)
	}
	return windows.UTF16ToString(buf[:size]), nil
}

// Read は lockfilePath のファイルを読み込み、コロン区切りの内容
// （"LeagueClient:<PID>:<port>:<password>:<protocol>"）を解析して
// Port と Password を格納した LCUAuth を返す。
func Read(lockfilePath string) (LCUAuth, error) {
	content, err := os.ReadFile(lockfilePath)
	if err != nil {
		return LCUAuth{}, fmt.Errorf("lockfile の読み込みに失敗: %w", err)
	}

	parts := strings.Split(strings.TrimSpace(string(content)), ":")
	if len(parts) < 5 {
		return LCUAuth{}, fmt.Errorf("lockfile の形式が不正です（フィールド数 %d）: %q", len(parts), string(content))
	}

	return LCUAuth{
		Port:     parts[2],
		Password: parts[3],
	}, nil
}
