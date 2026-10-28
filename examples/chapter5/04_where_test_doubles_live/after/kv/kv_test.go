package kv_test

import (
	"testing"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter5/04_where_test_doubles_live/after/kv"
	"github.com/holmeslin/learning-go-to-senior/examples/chapter5/04_where_test_doubles_live/after/kv/kvtest"
)

// ANCHOR: contract

// 同一組契約跑兩次：真的實作一次，附在套件裡的 fake 一次。
func TestDirStore(t *testing.T) {
	t.Parallel()
	kvtest.TestStore(t, func(t *testing.T) kvtest.Store { return kv.NewDirStore(t.TempDir()) })
}

func TestFake(t *testing.T) {
	t.Parallel()
	kvtest.TestStore(t, func(*testing.T) kvtest.Store { return kvtest.NewFake() })
}

// ANCHOR_END: contract
