// Package playlist 是一個小小的播放清單服務：GET /playlists/{id} 回傳清單的曲目與封面。
//
// 這是常見寫法：每個小函式各開一個 span，provider 從全域取得。
package playlist

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"go.opentelemetry.io/otel"
)

// ErrNotFound 表示沒有這份播放清單。
var ErrNotFound = errors.New("playlist not found")

// Store 是記憶體裡的資料表：播放清單對應曲目編號，曲目編號對應標題。
type Store struct {
	Playlists map[string][]string
	Titles    map[string]string
}

// TrackIDs 查某份播放清單的曲目編號；ctx 的位置留給真正的資料庫查詢。
func (s *Store) TrackIDs(_ context.Context, playlistID string) ([]string, bool) {
	ids, ok := s.Playlists[playlistID]
	return ids, ok
}

// Title 查一首曲目的標題。
func (s *Store) Title(_ context.Context, trackID string) string {
	return s.Titles[trackID]
}

// Service 處理播放清單的請求。
type Service struct {
	store    *Store
	client   *http.Client
	coverURL string
}

// NewService 建立 Service。
func NewService(store *Store, client *http.Client, coverURL string) *Service {
	return &Service{store: store, client: client, coverURL: coverURL}
}

// Handler 回傳掛好路由的 http.Handler。
func (s *Service) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /playlists/{id}", s.get)
	return mux
}

func (s *Service) get(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "get")
	defer span.End()

	body, err := s.render(ctx, r.PathValue("id"))
	switch {
	case errors.Is(err, ErrNotFound):
		http.Error(w, "playlist not found", http.StatusNotFound)
	case err != nil:
		http.Error(w, "internal error", http.StatusInternalServerError)
	default:
		io.WriteString(w, body)
	}
}

func (s *Service) cover(ctx context.Context, id string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.coverURL+"/covers/"+url.PathEscape(id), nil)
	if err != nil {
		return "", err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("cover service: status %d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	return string(b), err
}

// ANCHOR: spans

// tracer 從全域的 provider 取得；沒有人設定全域時，它什麼都不記錄。
var tracer = otel.Tracer("example.com/playlist")

func (s *Service) render(ctx context.Context, id string) (string, error) {
	ctx, span := tracer.Start(ctx, "render")
	defer span.End()

	ids, ok := s.store.TrackIDs(ctx, id)
	if !ok {
		return "", ErrNotFound
	}
	titles := make([]string, 0, len(ids))
	for _, trackID := range ids {
		titles = append(titles, s.title(ctx, trackID))
	}
	cover, err := s.cover(ctx, id)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s: %s (%s)", id, strings.Join(titles, ", "), cover), nil
}

func (s *Service) title(ctx context.Context, trackID string) string {
	ctx, span := tracer.Start(ctx, "title")
	defer span.End()

	return normalize(ctx, s.store.Title(ctx, trackID))
}

func normalize(ctx context.Context, title string) string {
	_, span := tracer.Start(ctx, "normalize")
	defer span.End()

	return strings.TrimSpace(title)
}

// ANCHOR_END: spans
