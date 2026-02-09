package err

// import (
// 	"errors"
// 	"fmt"
// 	"net/http"
// )

// type Type string

// const (
// 	NotFound      Type = "NOT_FOUND"
// 	Invalid       Type = "INVALID"
// 	Internal      Type = "INTERNAL"
// 	Unauthorized  Type = "UNAUTHORIZED"
// 	AlreadyExists Type = "ALREADY_EXISTS"
// )

// type Error struct {
// 	Type    Type
// 	Message string
// 	Err     error
// }

// func (e *Error) Error() string {
// 	if e.Err != nil {
// 		return fmt.Sprintf("%s: %v", e.Message, e.Err)
// 	}
// 	return e.Message
// }

// func (e *Error) Unwrap() error { return e.Err }

// func New(typ Type, msg string) error {
// 	return &Error{Type: typ, Message: msg}
// }

// func Wrap(err error, typ Type, msg string) error {
// 	return &Error{Type: typ, Message: msg, Err: err}
// }

// func HttpCode(err error) int {
// 	var ae *Error
// 	if errors.As(err, &ae) {
// 		switch ae.Type {
// 		case NotFound:
// 			return http.StatusNotFound
// 		case Invalid:
// 			return http.StatusBadRequest
// 		case Unauthorized:
// 			return http.StatusUnauthorized
// 		case AlreadyExists:
// 			return http.StatusConflict
// 		}
// 	}
// 	return http.StatusInternalServerError
// }