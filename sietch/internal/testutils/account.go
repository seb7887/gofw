package testutils

type Account struct {
	ID      int64 `db:"id"`
	Balance int   `db:"balance"`
}
