// Package lcu は LCU (League Client Update) のローカル API へのアクセスを提供する。
package lcu

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"

	"lol-game-tracker/internal/lockfile"
)

// 自己証明書を許可するクライアントを生成
func NewClient() *http.Client {
	timeout := 5 * time.Second
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
}

func get(client *http.Client, auth lockfile.LCUAuth, path string) ([]byte, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://127.0.0.1:%s%s", auth.Port, path), nil)
	if err != nil {
		return nil, fmt.Errorf("HTTPリクエストの作成に失敗: %w", err)
	}
	req.SetBasicAuth("riot", auth.Password)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTPリクエストの実行に失敗: %w", err)
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
