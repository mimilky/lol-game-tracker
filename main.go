package main

import (
	"fmt"
	"net/http"
	"os"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"lol-game-tracker/internal/lcu"
	"lol-game-tracker/internal/lockfile"
	"lol-game-tracker/internal/upload"
)

func run() error {
	lockfilePath, err := lockfile.Path()
	if err != nil {
		return err
	}
	auth, err := lockfile.Read(lockfilePath)
	if err != nil {
		return err
	}
	client := lcu.NewClient() //clientは使いまわし
	puuid, err := lcu.GetPUUID(client, auth)
	if err != nil {
		return err
	}

	matchHistory, err := lcu.GetMatchHistory(client, auth, puuid) //PUUIDの試合履歴取得
	if err != nil {
		return err
	}
	// 外部サーバ送信用のクライアント（LCU 用の InsecureSkipVerify クライアントは流用しない）
	uploadClient := &http.Client{Timeout: 10 * time.Second}

	participantsPerMatch := make(map[int64][]lcu.Participant, len(matchHistory))
	for _, gameID := range matchHistory {
		participants, err := lcu.GetMatchDetail(client, auth, gameID)
		if err != nil {
			return err
		}
		participantsPerMatch[gameID] = participants

		if err := upload.UploadMatchHistory(uploadClient, gameID, participants); err != nil {
			return err
		}
	}
	fmt.Println(participantsPerMatch)
	return nil
}

// MessageBox のフラグ（winuser.h）
const (
	mbOK              = 0x00000000
	mbIconError       = 0x00000010
	mbIconInformation = 0x00000040
	mbSetForeground   = 0x00010000 // 前面に出す
	mbTopMost         = 0x00040000 // 最前面
)

// showMessage は Windows のメッセージボックス（ポップアップ）を前面に表示する。
func showMessage(title, text string, iconFlag uint32) {
	user32 := windows.NewLazySystemDLL("user32.dll")
	messageBoxW := user32.NewProc("MessageBoxW")

	textPtr, _ := windows.UTF16PtrFromString(text)
	titlePtr, _ := windows.UTF16PtrFromString(title)
	messageBoxW.Call(
		0,
		uintptr(unsafe.Pointer(textPtr)),
		uintptr(unsafe.Pointer(titlePtr)),
		uintptr(mbOK|iconFlag|mbSetForeground|mbTopMost),
	)
}

func main() {
	if err := run(); err != nil {
		showMessage("エラー", err.Error(), mbIconError)
		os.Exit(1)
	}
	showMessage("完了", "処理が完了しました。", mbIconInformation)
}
