package errors

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

const (
	DatabaseError   = "database error"
	InternalError   = "internal error"
	ValidationError = "validation error"
	DataError       = "data error"
)

type AppError struct {
	Line     int
	File     string
	Err      error
	Debug    bool
	Category string
	Msg      string
	HumanMsg string
}

const unknownFile = "???"

func NewAppError(msg string) AppError {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = unknownFile
		line = 0
	}

	return AppError{
		Line: line,
		File: file,
		Msg:  msg,
	}
}

func NewAppErrorf(msg string, args ...any) AppError {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = unknownFile
		line = 0
	}

	return AppError{
		Line: line,
		File: file,
		Msg:  fmt.Sprintf(msg, args...),
	}
}

func Wrap(err error) AppError {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = unknownFile
		line = 0
	}

	return AppError{
		Line: line,
		File: file,
		Err:  err,
	}
}

func Wrapf(msg string, args ...any) AppError {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = unknownFile
		line = 0
	}

	return AppError{
		Line: line,
		File: file,
		Err:  fmt.Errorf(msg, args...), //nolint:err113
	}
}

func (e AppError) WithCategory(category string) AppError {
	return AppError{
		Line:     e.Line,
		File:     e.File,
		Err:      e.Err,
		Debug:    e.Debug,
		Category: category,
		Msg:      e.Msg,
		HumanMsg: e.HumanMsg,
	}
}

func (e AppError) WithHumanMsg(msg string) AppError {
	return AppError{
		Line:     e.Line,
		File:     e.File,
		Err:      e.Err,
		Debug:    e.Debug,
		Category: e.Category,
		Msg:      e.Msg,
		HumanMsg: msg,
	}
}

func (e AppError) Error() string {
	res := []string{}
	if e.Category != "" {
		res = append(res, e.Category+":")
	}

	if e.Msg != "" {
		res = append(res, e.Msg)
	}

	if e.Err != nil {
		res = append(res, e.Error())
	}

	if e.File == unknownFile {
		res = append(res, unknownFile)
	} else {
		res = append(res, fmt.Sprintf("(%s:%d)", e.File, e.Line))
	}

	return strings.Join(res, " ")
}

func (e AppError) Unwrap() error {
	return e.Err
}

func (e AppError) String() string {
	msg := e.HumanMsg
	if msg == "" {
		msg = e.Msg
	}

	if msg != "" {
		if e.Category != "" {
			return e.HumanMsg + " (" + e.Category + ")"
		}

		return e.HumanMsg
	}

	return e.Err.Error()
}

func AsAppError(err error) (AppError, bool) {
	var ae AppError
	if errors.As(err, &ae) {
		return ae, true
	}

	return ae, false
}
