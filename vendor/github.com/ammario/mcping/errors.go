package mcping

import (
	"errors"
	"fmt"
)

// ErrAddress indicates that the address format was bad.
type ErrAddress string

func (e ErrAddress) Error() string {
	return fmt.Sprintf("mcping: Could not parse address: got %s", string(e))
}

// ErrConnect indicates that the ping timed out.
type ErrConnect struct {
	inner error
}

func (e ErrConnect) Cause() error {
	return e.inner
}

func (e ErrConnect) Error() string {
	return fmt.Sprintf("mcping: Could not connect: %s", e.inner.Error())
}

// ErrVarint -> Could not decode varint
type ErrVarint struct {
	inner error
}

func (e ErrVarint) Cause() error {
	return e.inner
}

func (e ErrVarint) Error() string {
	return fmt.Sprintf("mcping: Could not decode varint: %s", e.inner.Error())
}

// ErrSmallPacket -> Response is too small
type ErrSmallPacket int

func (e ErrSmallPacket) Error() string {
	return fmt.Sprintf("mcping: Response too small (got %d bytes)", int(e))
}

// ErrBigPacket -> Response is too large
type ErrBigPacket int

func (e ErrBigPacket) Error() string {
	return fmt.Sprintf("mcping: Response too large (got %d bytes)", int(e))
}

// ErrPacketType -> Response packet incorrect
type ErrPacketType byte

func (e ErrPacketType) Error() string {
	return fmt.Sprintf("mcping: Response packet type incorrect (first byte was %X)", byte(e))
}

// ErrTimeout indicates that the ping timed out.
type ErrTimeout struct {
	inner error
}

func (e ErrTimeout) Cause() error {
	return e.inner
}

func (e ErrTimeout) Error() string {
	return fmt.Sprintf("mcping: Ping timed out: %s", e.inner.Error())
}
