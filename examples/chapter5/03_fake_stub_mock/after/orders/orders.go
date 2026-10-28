// Package orders 處理下單：算價、存檔、寄確認信。
package orders

import (
	"context"
	"errors"
	"fmt"
)

// ErrUnknownSKU 表示目錄裡沒有這個商品。
var ErrUnknownSKU = errors.New("unknown sku")

// Line 是訂單的一列：商品與數量。
type Line struct {
	SKU string
	Qty int
}

// Order 是已經算好價的訂單，Total 的單位是分。
type Order struct {
	ID       string // 存檔時配給的編號
	Customer string // 客戶的信箱
	Lines    []Line
	Total    int
}

// Catalog 查商品的單價（分）。商品不存在時回傳 ErrUnknownSKU。
type Catalog interface {
	Price(ctx context.Context, sku string) (int, error)
}

// Store 存訂單，回傳配給它的編號。
type Store interface {
	Save(ctx context.Context, o Order) (id string, err error)
}

// Mailer 寄信給客戶。
type Mailer interface {
	Send(ctx context.Context, to, subject string) error
}

// Service 處理下單的用例。
type Service struct {
	catalog Catalog
	store   Store
	mailer  Mailer
}

// NewService 回傳使用 catalog、store 與 mailer 的 Service。
func NewService(catalog Catalog, store Store, mailer Mailer) *Service {
	return &Service{catalog: catalog, store: store, mailer: mailer}
}

// Place 替 customer 下一張訂單：逐列查價、存檔，最後寄一封確認信。
func (s *Service) Place(ctx context.Context, customer string, lines []Line) (Order, error) {
	total := 0
	for _, l := range lines {
		price, err := s.catalog.Price(ctx, l.SKU)
		if err != nil {
			return Order{}, fmt.Errorf("price %s: %w", l.SKU, err)
		}
		total += price * l.Qty
	}
	return s.confirm(ctx, Order{Customer: customer, Lines: lines, Total: total})
}

// PlaceCached 是 Place 重構後的版本：同一個商品只查一次價格。
// 訂單與確認信都和 Place 一樣，只有呼叫 Catalog 的次數不同。
func (s *Service) PlaceCached(ctx context.Context, customer string, lines []Line) (Order, error) {
	prices := map[string]int{}
	total := 0
	for _, l := range lines {
		price, ok := prices[l.SKU]
		if !ok {
			var err error
			if price, err = s.catalog.Price(ctx, l.SKU); err != nil {
				return Order{}, fmt.Errorf("price %s: %w", l.SKU, err)
			}
			prices[l.SKU] = price
		}
		total += price * l.Qty
	}
	return s.confirm(ctx, Order{Customer: customer, Lines: lines, Total: total})
}

// confirm 存檔，成功後才寄確認信。
func (s *Service) confirm(ctx context.Context, o Order) (Order, error) {
	id, err := s.store.Save(ctx, o)
	if err != nil {
		return Order{}, fmt.Errorf("save order: %w", err)
	}
	o.ID = id
	if err := s.mailer.Send(ctx, o.Customer, "order "+id+" confirmed"); err != nil {
		return o, fmt.Errorf("send confirmation: %w", err)
	}
	return o, nil
}
