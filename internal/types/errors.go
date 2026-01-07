package types

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidObject = errors.New("invalid object")
	ErrServerError   = errors.New("server error")
	ErrNotFound      = errors.New("not found")
)

func NewErrInvalidObjectF(format string, objs ...any) error {
	return newErrorWithType(fmt.Errorf(format, objs...), ErrInvalidObject)
}

func NewErrServerF(format string, objs ...any) error {
	return newErrorWithType(fmt.Errorf(format, objs...), ErrServerError)
}

func NewErrNotFoundF(format string, objs ...any) error {
	return newErrorWithType(fmt.Errorf(format, objs...), ErrNotFound)
}

func newErrorWithType(err any, errorType error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %v", errorType, err)
}
