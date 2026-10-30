// Package playlist 是一個小小的播放清單服務：GET /playlists/{id} 回傳清單的曲目與封面。
//
// 這個套件只依賴 OpenTelemetry 的 API（trace、attribute、codes），不 import SDK，
// 用哪一種 TracerProvider 由呼叫端決定。
package playlist

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
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

// ANCHOR: entry

// Service 處理播放清單的請求。
type Service struct {
	tracer   trace.Tracer
	store    *Store
	client   *http.Client
	coverURL string
}

// NewService 建立 Service。tp 由呼叫端傳入：正式環境傳 SDK 的 provider，測試傳記憶體裡的 provider。
func NewService(tp trace.TracerProvider, store *Store, client *http.Client, coverURL string) *Service {
	return &Service{tracer: tp.Tracer("example.com/playlist"), store: store, client: client, coverURL: coverURL}
}

// Handler 回傳掛好路由的 http.Handler。
func (s *Service) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /playlists/{id}", s.get)
	return mux
}

func (s *Service) get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	// span 名稱用路由樣板，不用實際路徑；id 放進屬性。
	ctx, span := s.tracer.Start(r.Context(), r.Pattern,
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(attribute.String("playlist.id", id)))
	defer span.End()

	body, err := s.render(ctx, id)
	code := http.StatusOK
	switch {
	case errors.Is(err, ErrNotFound): // 呼叫端的錯，不是服務的錯，status 留在 Unset
		code = http.StatusNotFound
	case err != nil:
		code = http.StatusInternalServerError
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	span.SetAttributes(attribute.Int("http.response.status_code", code))
	if err != nil {
		http.Error(w, http.StatusText(code), code)
		return
	}
	io.WriteString(w, body)
}

// ANCHOR_END: entry

func (s *Service) render(ctx context.Context, id string) (string, error) {
	ids, err := s.trackIDs(ctx, id)
	if err != nil {
		return "", err
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

// ANCHOR: queries

func (s *Service) trackIDs(ctx context.Context, id string) ([]string, error) {
	ctx, span := s.tracer.Start(ctx, "SELECT playlists",
		trace.WithAttributes(attribute.String("playlist.id", id)))
	defer span.End()

	ids, ok := s.store.TrackIDs(ctx, id)
	if !ok {
		return nil, ErrNotFound // 查無資料不是查詢失敗，span 不記錄錯誤
	}
	return ids, nil
}

// title 每首曲目查一次，N 首就是 N 個同名的 SELECT tracks。
func (s *Service) title(ctx context.Context, trackID string) string {
	ctx, span := s.tracer.Start(ctx, "SELECT tracks",
		trace.WithAttributes(attribute.String("track.id", trackID)))
	defer span.End()

	return s.store.Title(ctx, trackID)
}

// ANCHOR_END: queries

// ANCHOR: cover

func (s *Service) cover(ctx context.Context, id string) (cover string, err error) {
	ctx, span := s.tracer.Start(ctx, "GET /covers/{id}",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attribute.String("http.request.method", http.MethodGet)))
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.coverURL+"/covers/"+url.PathEscape(id), nil)
	if err != nil {
		return "", err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	span.SetAttributes(attribute.Int("http.response.status_code", resp.StatusCode))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("cover service: status %d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	return string(b), err
}

// ANCHOR_END: cover
