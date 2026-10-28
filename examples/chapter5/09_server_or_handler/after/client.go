package notes

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// ErrNotFound 表示 server 回應沒有這則筆記。
var ErrNotFound = errors.New("note not found")

// Client 是筆記服務的 HTTP client。
type Client struct {
	BaseURL string       // 服務的位址，例如 "http://localhost:8080"
	HTTP    *http.Client // 送出請求用的 client
}

// Get 讀取筆記 id 的內容。筆記不存在時，錯誤會包住 ErrNotFound。
func (c *Client) Get(ctx context.Context, id string) (string, error) {
	text, err := c.get(ctx, id)
	if err != nil {
		return "", fmt.Errorf("get note %s: %w", id, err)
	}
	return text, nil
}

func (c *Client) get(ctx context.Context, id string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/notes/"+url.PathEscape(id), nil)
	if err != nil {
		return "", err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		body, err := io.ReadAll(resp.Body)
		return string(body), err
	case http.StatusNotFound:
		return "", ErrNotFound
	default:
		return "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
}
