package service

import "slices"

//
// mod.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

// DynamicCache holds values and create it when no exist.
type DynamicCache[T comparable, V any] struct {
	items   map[T]V
	creator func(key T) (V, error)
	used    []T
}

// GetOrCreate get value from cache or create it when no exists.
func (c *DynamicCache[T, V]) GetOrCreate(key T) (V, error) { //nolint:ireturn,nolintlint
	if value, ok := c.items[key]; ok {
		if !slices.Contains(c.used, key) {
			c.used = append(c.used, key)
		}

		return value, nil
	}

	value, err := c.creator(key)
	if err != nil {
		return *new(V), err
	}

	c.items[key] = value
	c.used = append(c.used, key)

	return value, nil
}

func (c *DynamicCache[T, V]) GetUsedValues() []V {
	res := make([]V, len(c.used))
	for i, v := range c.used {
		res[i] = c.items[v]
	}

	return res
}
