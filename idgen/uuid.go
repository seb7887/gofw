package idgen

import "github.com/google/uuid"

var _uuidGenerator = func() string {
	return uuid.New().String()
}

func NewUUID() string {
	return _uuidGenerator()
}

func UseUUID(fn func() string) {
	_uuidGenerator = fn
}
