package sietch

import "errors"

var (
	ErrItemNotFound         = errors.New("item not found")
	ErrItemAlreadyExists    = errors.New("item already exists")
	ErrNoUpdateItem         = errors.New("no item has been updated")
	ErrNoDeleteItem         = errors.New("no item has been deleted")
	ErrUnsupportedOperation = errors.New("unsupported operation")
)
