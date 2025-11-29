package main

//
// main.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	// _ "github.com/WAY29/icecream-go/icecream".

	cli "gitlab.com/kabes/go-gpo/internal/cli"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	cli.Main()
}
