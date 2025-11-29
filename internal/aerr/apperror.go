package aerr

import (
	"errors"
	"fmt"
	"io"
	"maps"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"github.com/rs/zerolog"
)

type AppError struct {
	err     error
	tags    []string
	msg     string
	userMsg string
	meta    map[string]any
	stack   []string
}

func NewWStack(msg string, args ...any) AppError {
	return AppError{
		stack: getStack(),
		msg:   fmt.Sprintf(msg, args...),
	}
}

func New(msg string, args ...any) AppError {
	return AppError{
		msg: fmt.Sprintf(msg, args...),
	}
}

func Newf(msg string, args ...any) AppError {
	return AppError{
		stack: getStack(),
		msg:   fmt.Sprintf(msg, args...),
	}
}

func Wrap(err error) AppError {
	return AppError{
		stack: getStack(),
		err:   err,
	}
}

func Wrapf(err error, msg string, args ...any) AppError {
	return AppError{
		stack: getStack(),
		err:   err,
		msg:   fmt.Sprintf(msg, args...),
	}
}

func (a AppError) WithMsg(msg string, args ...any) AppError {
	n := a.clone()
	n.msg = fmt.Sprintf(msg, args...)

	return n
}

func (a AppError) WithTag(tag string) AppError {
	if slices.Contains(a.tags, tag) {
		return a
	}

	n := a.clone()
	n.tags = append(n.tags, tag)

	return n
}

func (a AppError) WithUserMsg(msg string, args ...any) AppError {
	n := a.clone()
	n.userMsg = fmt.Sprintf(msg, args...)

	return n
}

func (a AppError) WithMeta(keyval ...any) AppError {
	if len(keyval)%2 != 0 {
		panic("invalid argument number to call WithMeta")
	}

	nerr := a.clone()

	if nerr.meta == nil {
		nerr.meta = make(map[string]any)
	}

	for i := 0; i < len(keyval); i += 2 {
		key, ok := keyval[i].(string)
		if !ok {
			key = fmt.Sprintf("%v", keyval[i])
		}

		nerr.meta[key] = keyval[i+1]
	}

	return nerr
}

// WithErr create copy of AppError with new error and updated stack.
func (a AppError) WithError(err error) AppError {
	n := a.clone()
	n.err = err
	n.stack = getStack()

	return n
}

func (a AppError) Is(target error) bool {
	tapperr, ok := target.(AppError)
	if !ok {
		return false
	}

	return tapperr.err == a.err &&
		tapperr.msg == a.msg &&
		tapperr.userMsg == a.userMsg &&
		slices.Compare(tapperr.tags, a.tags) == 0 &&
		slices.Compare(tapperr.stack, a.stack) == 0 &&
		maps.Equal(tapperr.meta, a.meta)
}

func (a AppError) Error() string {
	switch {
	case a.msg != "" && a.err != nil:
		return a.msg + "(" + a.err.Error() + ")"
	case a.msg != "":
		return a.msg
	case a.err != nil:
		return a.err.Error()
	default:
		return fmt.Sprintf("%v", a)
	}
}

func (a AppError) Unwrap() error {
	return a.err
}

func (a AppError) String() string {
	msg := a.userMsg
	if msg == "" {
		msg = a.msg
	}

	if msg != "" {
		return msg
	}

	return a.err.Error()
}

func (a AppError) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			msg := CollectErrors(a)
			fmt.Fprintf(s, "%+v\n", msg)

			return
		}

		fallthrough
	case 's', 'q':
		io.WriteString(s, a.Error())
	}
}

// Clone AppError, do not fill stack.
func (a AppError) clone() AppError {
	return AppError{
		stack:   a.stack,
		msg:     a.msg,
		tags:    slices.Clone(a.tags),
		userMsg: a.userMsg,
		meta:    maps.Clone(a.meta),
		err:     a.err,
	}
}

//-------------------------------------------------------------

// ApplyFor create copy of AppError with replaced error and updated location.
// Optional arguments set msg and userMsg (if are not empty).
func ApplyFor(aerr AppError, err error, msg ...string) AppError {
	if err == nil {
		panic("err for apply is nil")
	}

	nerr := aerr.clone()
	nerr.stack = getStack()
	nerr.err = err

	if len(msg) == 0 {
		return nerr
	}

	if msg[0] != "" {
		nerr.msg = msg[0]
	}

	if len(msg) > 1 && msg[1] != "" {
		nerr.userMsg = msg[1]
	}

	return nerr
}

//-------------------------------------------------------------

func AsAppError(err error) (AppError, bool) {
	var ae AppError
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
		if ae, ok := err.(AppError); ok { //nolint:errorlint
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

func Flatten(err error) []AppError {
	errs := []AppError{}

	for ; err != nil; err = errors.Unwrap(err) {
		if ae, ok := err.(AppError); ok { //nolint:errorlint
			errs = append(errs, ae)
		}
	}

	slices.Reverse(errs)

	return errs
}

func CollectErrors(err error) []string {
	errs := []string{}

	for ; err != nil; err = errors.Unwrap(err) {
		apperr, ok := err.(AppError) //nolint:errorlint
		if !ok {
			errs = append(errs, err.Error())

			continue
		}

		errmsg := apperr.Error()

		if len(apperr.stack) > 0 {
			errmsg += " [" + apperr.stack[0] + "]"
		}

		errmsg += fmt.Sprintf("%v/%v", apperr.tags, apperr.meta)

		errs = append(errs, errmsg)
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
		stack, errs []string
		meta        map[string]any
	)

	usermsg := make(uniqueList, 0)
	tags := make(uniqueList, 0)

	for err := m.err; err != nil; err = errors.Unwrap(err) {
		if apperr, ok := err.(AppError); ok { //nolint:errorlint,nestif
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
		} else {
			errs = append(errs, err.Error())
		}
	}

	if len(usermsg) > 0 {
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

	if len(tags) > 0 {
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
