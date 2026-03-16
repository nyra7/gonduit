package session

import (
	"crypto/rand"
	"errors"
	"fmt"
	"shared/proto"
	"time"

	"github.com/oklog/ulid/v2"
)

func newID() string {
	t := time.Now()
	entropy := ulid.Monotonic(rand.Reader, 0)
	return ulid.MustNew(ulid.Timestamp(t), entropy).String()
}

func expect[T any](msg interface{}) (T, error) {

	// Try to cast to the expected type
	if v, ok := msg.(T); ok {
		return v, nil
	}

	// Try to cast to the server error type
	if errMsg, ok := msg.(*proto.ShellServerMessage_Error); ok {
		var zero T
		return zero, errors.New(errMsg.Error.Error)
	}

	// Unexpected message type
	var zero T
	return zero, fmt.Errorf("invalid message type: expected %T, got %T", zero, msg)

}
