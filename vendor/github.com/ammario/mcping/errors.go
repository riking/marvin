package mcping

import (
	"errors"
)

//ErrAddress -> Could not parse address
var ErrAddress = errors.New("mcping: could not parse address")

//ErrResolve -. Could not resolve address
var ErrResolve = errors.New("mcping: Could not resolve address")

//ErrConnect -> Could not connect to host
var ErrConnect = errors.New("mcping: Could not connect to host")

//ErrVarint -> Could not decode varint
var ErrVarint = errors.New("mcping: Could not decode varint")

//ErrSmallPacket -> Response is too small
var ErrSmallPacket = errors.New("mcping: Response too small")

//ErrBigPacket -> Response is too large
var ErrBigPacket = errors.New("mcping: Response too large")

//ErrPacketType -> Response packet incorrect
var ErrPacketType = errors.New("mcping: Response packet type incorrect")

//ErrTimeout -> Timeout error
var ErrTimeout = errors.New("mcping: Timeout occured")
