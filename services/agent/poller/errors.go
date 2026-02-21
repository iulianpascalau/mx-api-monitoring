package poller

import "net/http"

type errStatusNotOK int

func (e errStatusNotOK) Error() string {
	return "non-2xx HTTP status code: " + http.StatusText(int(e))
}

type errPathNotFound string

func (e errPathNotFound) Error() string {
	return "JSON path not found in response: " + string(e)
}
