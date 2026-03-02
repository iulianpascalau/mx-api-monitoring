package notifiers

import "errors"

var (
	errReturnCodeIsNotOk = errors.New("HTTP return code is not OK")
	errNilLogger         = errors.New("nil logger")
)
