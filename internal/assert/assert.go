package assert

//
// mod.go
// based on https://antonz.org/do-not-testify/
//

import (
	"bytes"
	"cmp"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// Equal asserts that got is equal to want.
func Equal[T any](tb testing.TB, got, want T) bool {
	tb.Helper()

	if !areEqual(got, want) {
		tb.Errorf("got: %#v; want: %#v", got, want)

		return false
	}

	return true
}

// NotEqual asserts that got is no equal to want.
func NotEqual[T any](tb testing.TB, got, want T) bool {
	tb.Helper()

	if areEqual(got, want) {
		tb.Errorf("got: %#v; want other values", got)

		return false
	}

	return true
}

// NoErr asserts that the got error is nil.
func NoErr(tb testing.TB, got error) bool {
	tb.Helper()

	if got != nil {
		tb.Errorf("got unexpected error: %#+v", got)

		return false
	}

	return true
}

// Err asserts that the got error matches the want.
func Err(tb testing.TB, got error) bool {
	tb.Helper()

	if got == nil {
		tb.Error("got: <nil>; want: error")

		return false
	}

	return true
}

// Err asserts that the got error matches the want.
func ErrSpec(tb testing.TB, got error, want any) bool {
	tb.Helper()

	// We'll only match against the first want for simplicity.
	if got == nil {
		tb.Errorf("got: <nil>; want: %v", want)

		return false
	}

	switch wanttype := want.(type) {
	case string:
		if !strings.Contains(got.Error(), wanttype) {
			tb.Errorf("got: %q; want: %q", got.Error(), wanttype)

			return false
		}
	case error:
		if !errors.Is(got, wanttype) {
			tb.Errorf("got: %T(%v); want: %T(%v)", got, got, wanttype, wanttype)

			return false
		}
	case reflect.Type:
		target := reflect.New(wanttype).Interface()
		if !errors.As(got, target) {
			tb.Errorf("got: %T; want: %s", got, wanttype)

			return false
		}
	default:
		tb.Errorf("unsupported want type: %T", want)
	}

	return true
}

// True asserts that got is true.
func True(tb testing.TB, got bool) bool {
	tb.Helper()

	if !got {
		tb.Error("got: false; want: true")
	}

	return got
}

// equaler is an interface for types with an Equal method
// (like time.Time or net.IP).
type equaler[T any] interface {
	Equal(other T) bool
}

// areEqual checks if a and b are equal.
func areEqual[T any](val1, val2 T) bool {
	// Check if both are nil.
	if isNil(val1) && isNil(val2) {
		return true
	}

	// Try to compare using an Equal method.
	if eq, ok := any(val1).(equaler[T]); ok {
		return eq.Equal(val2)
	}

	// Special case for byte slices.
	if aBytes, ok := any(val1).([]byte); ok {
		if bBytes, ok := any(val2).([]byte); ok {
			return bytes.Equal(aBytes, bBytes)
		}
	}
	// Fallback to reflective comparison.
	return reflect.DeepEqual(val1, val2)
}

// isNil checks if v is nil.
func isNil(v any) bool {
	if v == nil {
		return true
	}
	// A non-nil interface can still hold a nil value,
	// so we must check the underlying value.
	rv := reflect.ValueOf(v)

	switch rv.Kind() { //nolint:exhaustive
	case reflect.Chan,
		reflect.Func,
		reflect.Interface,
		reflect.Map,
		reflect.Pointer,
		reflect.Slice,
		reflect.UnsafePointer:
		return rv.IsNil()
	default:
		return false
	}
}

// EqualSorted asserts that got sorted slice is equal to want sorted slice.
func EqualSorted[S ~[]E, E cmp.Ordered](tb testing.TB, got, want S) bool {
	tb.Helper()

	got = slices.Clone(got)
	slices.Sort(got)

	want = slices.Clone(want)
	slices.Sort(want)

	if !slices.Equal(got, want) {
		tb.Errorf("got: %#v; want: %#v", got, want)

		return false
	}

	return true
}
