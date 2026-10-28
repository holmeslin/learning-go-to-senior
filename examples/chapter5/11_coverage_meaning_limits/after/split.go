// Package split 把一筆金額依權重的比例分成幾份，例如幾個人分攤一張帳單。
package split

import (
	"errors"
	"fmt"
)

var (
	// ErrNoWeights 表示沒有給任何權重。
	ErrNoWeights = errors.New("no weights")
	// ErrBadWeight 表示有權重不是正數。
	ErrBadWeight = errors.New("weight must be positive")
	// ErrNegativeTotal 表示要分的金額是負數。
	ErrNegativeTotal = errors.New("total must not be negative")
)

// Split 依 weights 的比例把 total（單位：分）分成 len(weights) 份，總和一定等於 total。
// 每份無條件捨去，剩下的不到 len(weights) 分全部歸第一份。
// 沒有權重、權重不是正數、total 是負數時回傳錯誤。
func Split(total int, weights ...int) ([]int, error) {
	if len(weights) == 0 {
		return nil, ErrNoWeights
	}
	if total < 0 {
		return nil, fmt.Errorf("split %d: %w", total, ErrNegativeTotal)
	}
	sum := 0
	for _, w := range weights {
		if w <= 0 {
			return nil, ErrBadWeight
		}
		sum += w
	}
	shares := make([]int, len(weights))
	left := total
	for i, w := range weights {
		shares[i] = total * w / sum
		left -= shares[i]
	}
	shares[0] += left
	return shares, nil
}
