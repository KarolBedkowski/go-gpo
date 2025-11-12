package service

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
}

// GetOrCreate get value from cache or create it when no exists.
func (c *DynamicCache[T, V]) GetOrCreate(key T) (V, error) {
	if value, ok := c.items[key]; ok {
		return value, nil
	}

	value, err := c.creator(key)
	if err != nil {
		return *new(V), err
	}

	c.items[key] = value

	return value, nil
}
