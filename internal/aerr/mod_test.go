package aerr

//
// mod_test.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"errors"
	"testing"

	"gitlab.com/kabes/go-gpo/internal/assert"
)

func TestUniqueList(t *testing.T) {
	var ulist uniqueList

	assert.Equal(t, ulist, nil)

	// add new
	ulist.append("a")
	assert.Equal(t, len(ulist), 1)
	assert.Equal(t, ulist[0], "a")

	// add new
	ulist.append("b", "c")
	assert.Equal(t, len(ulist), 3)
	assert.Equal(t, ulist[0], "a")
	assert.Equal(t, ulist[1], "b")
	assert.Equal(t, ulist[2], "c")

	// a exists
	ulist.append("a")
	assert.Equal(t, len(ulist), 3)
	assert.Equal(t, ulist[0], "a")
	assert.Equal(t, ulist[1], "b")
	assert.Equal(t, ulist[2], "c")

	// b exists, add d
	ulist.append("b", "d")
	assert.Equal(t, len(ulist), 4)
	assert.Equal(t, ulist[0], "a")
	assert.Equal(t, ulist[1], "b")
	assert.Equal(t, ulist[2], "c")
	assert.Equal(t, ulist[3], "d")

	// no changes
	ulist.append("b", "d")
	assert.Equal(t, len(ulist), 4)
	assert.Equal(t, ulist[0], "a")
	assert.Equal(t, ulist[1], "b")
	assert.Equal(t, ulist[2], "c")
	assert.Equal(t, ulist[3], "d")
}

func TestAppErrorWrap(t *testing.T) {
	err := errors.New("error1")

	aerr1 := Wrap(err)
	assert.True(t, errors.Is(aerr1, err))
	assert.Equal(t, errors.Unwrap(aerr1), err)
	assert.True(t, aerr1.stack != nil)
	assert.Equal(t, aerr1.String(), "error1")
}

func TestAppErrorMsg(t *testing.T) {
	err := errors.New("error1")

	aerr0 := Wrap(err)
	aerr1 := aerr0.WithMsg("apperror%d", 1)
	assert.True(t, aerr1 != aerr0)
	assert.Equal(t, aerr1.stack, aerr0.stack)
	assert.True(t, errors.Is(aerr1, err))
	assert.Equal(t, aerr1.msg, "apperror1")
	assert.Equal(t, aerr1.String(), "apperror1")

	assert.Equal(t, GetUserMessage(aerr1), "")
	assert.Equal(t, GetUserMessageOr(aerr1, "--"), "--")

	aerr2 := aerr1.WithUserMsg("user message %d", 123)
	assert.True(t, aerr2 != nil)
	assert.True(t, aerr2 != aerr1)
	assert.True(t, errors.Is(aerr2, err))
	assert.Equal(t, aerr2.stack, aerr0.stack)
	assert.Equal(t, aerr2.msg, "apperror1")
	assert.Equal(t, aerr2.String(), "user message 123")

	assert.Equal(t, GetUserMessage(aerr2), "user message 123")
	assert.Equal(t, GetUserMessageOr(aerr2, "--"), "user message 123")
}

func TestAppErrorMeta(t *testing.T) {
	err := errors.New("error1")

	aerr0 := Wrap(err)
	aerr1 := aerr0.WithMeta("k1", 1, "k2", "v2")
	assert.Equal(t, len(aerr1.meta), 2)
	assert.Equal(t, aerr1.meta["k1"], 1)
	assert.Equal(t, aerr1.meta["k2"], "v2")

	// 22 key should be converted to str
	aerr2 := aerr1.WithMeta("k1", 2, "k3", "v3", 22, "v22")
	assert.Equal(t, len(aerr2.meta), 4)
	assert.Equal(t, aerr2.meta["k1"], 2)
	assert.Equal(t, aerr2.meta["k2"], "v2")
	assert.Equal(t, aerr2.meta["k3"], "v3")
	assert.Equal(t, aerr2.meta["22"], "v22")
	// no changes in aerr1
	assert.Equal(t, len(aerr1.meta), 2)
}

func TestAppErrorTags(t *testing.T) {
	aerr0 := New("error1")

	aerr1 := aerr0.WithTag("k1")
	assert.Equal(t, GetTags(aerr1), []string{"k1"})

	aerr1 = aerr1.WithTag("k2")
	assert.Equal(t, GetTags(aerr1), []string{"k1", "k2"})
	assert.True(t, HasTag(aerr1, "k1"))
	assert.True(t, HasTag(aerr1, "k2"))
	assert.True(t, !HasTag(aerr1, "k3"))

	aerr2 := aerr1.WithTag("k3")
	assert.Equal(t, GetTags(aerr2), []string{"k1", "k2", "k3"})
	assert.Equal(t, GetTags(aerr1), []string{"k1", "k2"})
	assert.True(t, HasTag(aerr2, "k1"))
	assert.True(t, HasTag(aerr2, "k2"))
}

func TestAppErrorErr(t *testing.T) {
	err := NewSimple("simple error%d", 1)
	err0 := Newf("error %s-%d", "1", 2)

	aerr1 := err.WithError(err0)
	assert.True(t, aerr1.err == err0)
	assert.True(t, aerr1.Unwrap() == err0)
	assert.Equal(t, aerr1.String(), "simple error1")
	// new stack
	assert.NotEqual(t, aerr1.stack, err0.stack)
	// getstack return stack from deepest error
	assert.Equal(t, GetStack(aerr1), err0.stack)
}
