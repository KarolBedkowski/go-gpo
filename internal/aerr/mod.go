package aerr

import (
	"errors"
	"fmt"
	"maps"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"github.com/rs/zerolog"
)

const (
	InternalError   = "internal error"
	ValidationError = "validation error"
	DataError       = "data error"
)

type AppError struct {
	err     error
	tags    []string
	msg     string
	userMsg string
	meta    map[string]any
	stack   []string
}

func New(msg string) *AppError {
	return &AppError{
		stack: getStack(),
		msg:   msg,
	}
}

func NewSimple(msg string) *AppError {
	return &AppError{msg: msg}
}

func Newf(msg string, args ...any) *AppError {
	return &AppError{
		stack: getStack(),
		msg:   fmt.Sprintf(msg, args...),
	}
}

func Wrap(err error) *AppError {
	if err == nil {
		return nil
	}

	return &AppError{
		stack: getStack(),
		err:   err,
	}
}

func Wrapf(err error, msg string, args ...any) *AppError {
	if err == nil {
		return nil
	}

	return &AppError{
		stack: getStack(),
		err:   err,
		msg:   fmt.Sprintf(msg, args...),
	}
}

func (a *AppError) WithMsg(msg string) *AppError {
	if a == nil {
		return nil
	}

	a.msg = msg

	return a
}

func (a *AppError) WithTag(tag string) *AppError {
	if a == nil {
		return nil
	}

	if slices.Contains(a.tags, tag) {
		return a
	}

	a.tags = append(a.tags, tag)

	return a
}

func (a *AppError) WithUserMsg(msg string) *AppError {
	if a == nil {
		return nil
	}

	a.userMsg = msg

	return a
}

func (a *AppError) WithMeta(keyval ...any) *AppError {
	if a == nil {
		return nil
	}

	if len(keyval)%2 != 0 {
		panic("invalid argument number to call WithMeta")
	}

	if a.meta == nil {
		a.meta = make(map[string]any)
	}

	for i := 0; i < len(keyval); i += 2 {
		key, ok := keyval[i].(string)
		if !ok {
			key = fmt.Sprintf("%v", keyval[i])
		}

		a.meta[key] = keyval[i]
	}

	return a
}

func (a *AppError) WithError(err error) *AppError {
	if err == nil {
		return nil
	}

	a.err = err

	return a
}

func (a *AppError) Error() string {
	if a == nil {
		return ""
	}

	if a.msg != "" {
		return a.msg
	}

	return a.err.Error()
}

func (a *AppError) Unwrap() error {
	if a == nil {
		return nil
	}

	return a.err
}

func (a *AppError) String() string {
	if a == nil {
		return ""
	}

	msg := a.userMsg
	if msg == "" {
		msg = a.msg
	}

	if msg != "" {
		return a.userMsg
	}

	return a.err.Error()
}

// Clone AppError, update location.
func (a *AppError) Clone() *AppError {
	if a == nil {
		return nil
	}

	return &AppError{
		stack:   getStack(),
		msg:     a.msg,
		tags:    slices.Clone(a.tags),
		userMsg: a.userMsg,
		meta:    maps.Clone(a.meta),
		err:     a.err,
	}
}

//-------------------------------------------------------------

// ApplyFor create copy of AppError with replaced error and updated location.
func ApplyFor(aerr *AppError, err error) *AppError {
	if err == nil {
		return nil
	}

	return &AppError{
		stack:   getStack(),
		msg:     aerr.msg,
		tags:    slices.Clone(aerr.tags),
		userMsg: aerr.userMsg,
		meta:    maps.Clone(aerr.meta),
		err:     err,
	}
}

//-------------------------------------------------------------

func AsAppError(err error) (*AppError, bool) {
	var ae *AppError
	if errors.As(err, &ae) {
		return ae, true
	}

	return ae, false
}

//-------------------------------------------------------------

func HasTag(err error, tag string) bool {
	for _, ae := range Flatten(err) {
		if slices.Contains(ae.tags, tag) {
			return true
		}
	}

	return false
}

func GetTags(err error) []string {
	tags := []string{}

	for _, ae := range Flatten(err) {
		for _, t := range ae.tags {
			if !slices.Contains(tags, t) {
				tags = append(tags, t)
			}
		}
	}

	return tags
}

func GetUserMessages(err error) []string {
	msgs := []string{}

	for _, ae := range Flatten(err) {
		if ae.userMsg != "" {
			msgs = append(msgs, ae.userMsg)
		}
	}

	return msgs
}

func GetUserMessage(err error) string {
	for _, ae := range Flatten(err) {
		if ae.userMsg != "" {
			return ae.userMsg
		}
	}

	return ""
}

func GetUserMessageOr(err error, defaultmsg string) string {
	msg := GetUserMessage(err)
	if msg == "" {
		return defaultmsg
	}

	return msg
}

func GetStack(err error) []string {
	for _, ae := range Flatten(err) {
		if len(ae.stack) > 0 {
			return ae.stack
		}
	}

	return nil
}

func GetErrors(err error) []string {
	errs := []string{}

	for err != nil {
		if ae, ok := err.(*AppError); ok { //nolint:errorlint
			if ae.msg != "" {
				errs = append(errs, ae.msg)
			}
		} else {
			errs = append(errs, err.Error())
		}

		err = errors.Unwrap(err)
	}

	slices.Reverse(errs)

	return errs
}

func Flatten(err error) []*AppError {
	errs := []*AppError{}

	for ; err != nil; err = errors.Unwrap(err) {
		if ae, ok := err.(*AppError); ok { //nolint:errorlint
			errs = append(errs, ae)
		}
	}

	slices.Reverse(errs)

	return errs
}

//-------------------------------------------------------------

type uniqueList []string

func (u *uniqueList) append(value ...string) {
	for _, v := range value {
		if !slices.Contains(*u, v) {
			*u = append(*u, v)
		}
	}
}

//-------------------------------------------------------------

type zerologErrorMarshaller struct {
	err error
}

func (m zerologErrorMarshaller) MarshalZerologObject(event *zerolog.Event) { //nolint:cyclop
	var (
		stack, errs   []string
		usermsg, tags uniqueList
		meta          map[string]any
	)

	for err := m.err; err != nil; err = errors.Unwrap(err) {
		apperr, ok := err.(*AppError) //nolint:errorlint
		if !ok {
			errs = append(errs, err.Error())

			continue
		}

		if apperr.userMsg != "" {
			usermsg.append(apperr.userMsg)
		}

		if apperr.stack != nil {
			stack = apperr.stack
		}

		if apperr.msg != "" {
			errs = append(errs, apperr.msg)
		}

		if apperr.tags != nil {
			tags.append(apperr.tags...)
		}

		if apperr.meta != nil {
			if meta == nil {
				meta = make(map[string]any)
			}

			maps.Copy(meta, apperr.meta)
		}
	}

	if usermsg != nil {
		slices.Reverse(usermsg)
		event.Strs("user_msg", usermsg)
	}

	if stack != nil {
		event.Strs("stack", stack)
	}

	if errs != nil {
		slices.Reverse(errs)
		event.Strs("errors", errs)
	}

	if tags != nil {
		event.Strs("tags", tags)
	}

	if meta != nil {
		event.Any("meta", meta)
	}
}

func ErrorMarshalFunc(err error) any {
	if err != nil {
		return zerologErrorMarshaller{err}
	}

	return err
}

//-------------------------------------------------------------

// func getLocation() string {
// 	_, file, line, ok := runtime.Caller(2) //nolint:mnd
// 	if ok {
// 		return fmt.Sprintf("%s:%d", file, line)
// 	}

// 	return ""
// }

var skipFunctions = []string{
	"net/http.HandlerFunc.ServeHTTP",
	"runtime.goexit",
}

const maxStack = 10

func getStack() []string {
	pc := make([]uintptr, 32) //nolint:mnd

	n := runtime.Callers(3, pc) //nolint:mnd
	if n == 0 {
		return nil
	}

	pc = pc[:n]
	frames := runtime.CallersFrames(pc)
	stack := make([]string, 0, n)

	for {
		frame, more := frames.Next()
		funcname := frame.Func.Name()

		if !slices.Contains(skipFunctions, funcname) {
			funcname = funcname[strings.LastIndex(funcname, "/")+1:]
			funcname = funcname[strings.Index(funcname, ".")+1:]
			stack = append(stack, frame.File+":"+strconv.Itoa(frame.Line)+":"+funcname)
		}

		if !more || len(stack) == maxStack {
			break
		}
	}

	return stack
}
