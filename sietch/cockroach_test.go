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

func TestCockroachDBConnector_getValues(t *testing.T) {
	conn, err := NewCockroachDBConnector[testutils.Account, int64](
		nil,
		"test",
		func(t *testutils.Account) int64 {
			return t.ID
		})

	if err != nil {
		t.Errorf("NewCockroachDBConnector returned error: %s", err)
	}

	item := &testutils.Account{
		ID:      1,
		Balance: 100,
	}
	values, err := conn.getValues(item)
	if err != nil {
		t.Errorf("getValues returned error: %s", err)
	}

	if len(values) != 2 {
		t.Errorf("getValues returned %d values, expected 2", len(values))
	}
}

func TestCockroachDBConnector_getScanDestinations(t *testing.T) {
	conn, err := NewCockroachDBConnector[testutils.Account, int64](
		nil,
		"test",
		func(t *testutils.Account) int64 {
			return t.ID
		})

	if err != nil {
		t.Errorf("NewCockroachDBConnector returned error: %s", err)
	}

	item := &testutils.Account{
		ID:      1,
		Balance: 100,
	}

	d, err := conn.getScanDestinations(item)
	if err != nil {
		t.Errorf("getScanDestinations returned error: %s", err)
	}

	if len(d) != 2 {
		t.Errorf("getScanDestinations returned %d destinations, expected 2", len(d))
	}
}
