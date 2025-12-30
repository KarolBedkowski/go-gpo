package sqlite

//
// sqlite_test.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"fmt"
	"testing"

	"gitlab.com/kabes/go-gpo/internal/assert"
)

func TestPrepareSqliteConnstr(t *testing.T) {
	tests := []struct {
		connstr  string
		expected string
		experr   bool
	}{
		{"", "", true},
		{"?abc?_fk=1", "", true},
		{"/abc/abc?_fk=1", "/abc/abc?_fk=1", false},
		{"/abc/abc?_fk=0", "/abc/abc?_fk=0", false},
		{"/abc/abc?__foreign_keys=ON", "/abc/abc?__foreign_keys=ON", false},
		{"/abc/abc", "/abc/abc?_fk=ON", false},
		{"/abc/abc?_abc=123", "/abc/abc?_abc=123&_fk=ON", false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt), func(t *testing.T) {
			res, err := prepareSqliteConnstr(tt.connstr)
			if tt.experr {
				assert.Err(t, err)
			} else {
				assert.NoErr(t, err)
				assert.Equal(t, res, tt.expected)
			}
		})
	}
}
