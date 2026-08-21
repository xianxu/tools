package store_test

import (
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
	"github.com/xianxu/tools/cmd/define/store/storetest"
)

func TestMemConformance(t *testing.T) {
	storetest.Suite(t, func(t *testing.T) store.Store { return store.NewMem() })
}
