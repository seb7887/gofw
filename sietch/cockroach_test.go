package sietch

import (
	"github.com/seb7887/gofw/sietch/internal/testutils"
	"testing"
)

func TestNewCockroachDBConnector(t *testing.T) {
	conn, err := NewCockroachDBConnector[testutils.Account, int64](
		nil,
		"test",
		func(t *testutils.Account) int64 {
			return t.ID
		})

	if err != nil {
		t.Errorf("NewCockroachDBConnector returned error: %s", err)
	}

	if len(conn.columns) != 2 {
		t.Errorf("NewCockroachDBConnector returned %d columns, expected 2", len(conn.columns))
	}
}
