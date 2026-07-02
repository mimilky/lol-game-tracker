package main

import (
	"fmt"
	"os"

	"lol-game-tracker/internal/lcu"
	"lol-game-tracker/internal/lockfile"
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
	puuidsPerMatch := make(map[int64][]string, len(matchHistory))
	for _, gameID := range matchHistory {
		puuids, err := lcu.GetMatchDetail(client, auth, gameID)
		if err != nil {
			return err
		}
		puuidsPerMatch[gameID] = puuids
	}
	fmt.Println(puuidsPerMatch)
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "エラー:", err)
		os.Exit(1)
	}
}
