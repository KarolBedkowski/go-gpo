package errors

import (
	"fmt"
	"runtime"
)

type AppError struct {
	Line  int
	File  string
	Err   error
	Debug bool
}

func NewAppError(err error) *AppError {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "???"
		line = 0
	}

	return &AppError{
		Line: line,
		File: file,
		Err:  err,
	}
}

func (e *AppError) Error() string {
	return fmt.Sprintf("%s (%s:%d)", e.Err, e.File, e.Line)
}

func (e *AppError) Unwrap() error {
	return e.Err
}
