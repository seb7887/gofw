package idgen

import (
	"github.com/oklog/ulid/v2"
	"math/rand"
	"time"
)

var _ulidGenerator = func() string {
	var (
		entropy = rand.New(rand.NewSource(time.Now().UnixNano()))
		ms      = ulid.Timestamp(time.Now())
		id, _   = ulid.New(ms, entropy)
	)
	return id.String()
}

func NewULID() string {
	return _ulidGenerator()
}

func UseULID(fn func() string) {
	_ulidGenerator = fn
}
