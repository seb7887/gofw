package sietch

import (
	"context"
	"testing"

	"github.com/seb7887/gofw/sietch/internal/testutils"
)

func TestCockroachDBTransactionalInterface(t *testing.T) {
	t.Run("CockroachDBConnector implements Transactional", func(t *testing.T) {
		// This test verifies that the type assertion works
		repo := &CockroachDBConnector[testutils.Account, int64]{}

		_, ok := interface{}(repo).(Transactional[testutils.Account, int64])
		if !ok {
			t.Error("CockroachDBConnector should implement Transactional interface")
		}
	})
}

func TestInMemoryTransactionalInterface(t *testing.T) {
	t.Run("InMemoryConnector implements Transactional", func(t *testing.T) {
		repo := NewInMemoryConnector[testutils.Account, int64](
			func(a *testutils.Account) int64 { return a.ID },
		)

		_, ok := interface{}(repo).(Transactional[testutils.Account, int64])
		if !ok {
			t.Error("InMemoryConnector should implement Transactional interface")
		}
	})
}

func TestRedisTransactionalInterface(t *testing.T) {
	ctx := context.Background()

	t.Run("RedisConnector returns ErrUnsupportedOperation for WithTx", func(t *testing.T) {
		// Create a minimal Redis connector (will fail on actual operations but that's OK for this test)
		repo := &RedisConnector[testutils.Account, int64]{}

		err := repo.WithTx(ctx, func(txRepo Repository[testutils.Account, int64]) error {
			return nil
		})

		if err != ErrUnsupportedOperation {
			t.Errorf("Expected ErrUnsupportedOperation, got %v", err)
		}
	})

	t.Run("RedisConnector returns ErrUnsupportedOperation for Count", func(t *testing.T) {
		repo := &RedisConnector[testutils.Account, int64]{}

		filter := &Filter{}
		count, err := repo.Count(ctx, filter)

		if err != ErrUnsupportedOperation {
			t.Errorf("Expected ErrUnsupportedOperation, got %v", err)
		}
		if count != 0 {
			t.Errorf("Expected count 0, got %d", count)
		}
	})
}

func TestTransactionHelpers(t *testing.T) {
	t.Run("contains helper function", func(t *testing.T) {
		tests := []struct {
			s      string
			substr string
			want   bool
		}{
			{"hello world", "world", true},
			{"hello world", "foo", false},
			{"", "", true},
			{"hello", "", true},
			{"", "hello", false},
		}

		for _, tt := range tests {
			got := contains(tt.s, tt.substr)
			if got != tt.want {
				t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		}
	})

	t.Run("findSubstring helper function", func(t *testing.T) {
		tests := []struct {
			s      string
			substr string
			want   bool
		}{
			{"hello world", "world", true},
			{"hello world", "foo", false},
			{"abcdef", "cde", true},
			{"abcdef", "xyz", false},
		}

		for _, tt := range tests {
			got := findSubstring(tt.s, tt.substr)
			if got != tt.want {
				t.Errorf("findSubstring(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		}
	})

	t.Run("joinString helper function", func(t *testing.T) {
		tests := []struct {
			slice []string
			sep   string
			want  string
		}{
			{[]string{}, ",", ""},
			{[]string{"a"}, ",", "a"},
			{[]string{"a", "b"}, ",", "a,b"},
			{[]string{"a", "b", "c"}, ", ", "a, b, c"},
		}

		for _, tt := range tests {
			got := joinString(tt.slice, tt.sep)
			if got != tt.want {
				t.Errorf("joinString(%v, %q) = %q, want %q", tt.slice, tt.sep, got, tt.want)
			}
		}
	})
}
