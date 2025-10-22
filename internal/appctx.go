package internal

import "context"

var CtxUserKey = any("CtxUserKey")

func ContextUser(ctx context.Context) string {
	suser, ok := ctx.Value(CtxUserKey).(string)
	if ok {
		return suser
	}

	return ""
}

func ContextWithUser(ctx context.Context, username string) context.Context {
	return context.WithValue(ctx, CtxUserKey, username)
}

// ------------------------------------------------------

var CtxDeviceKey = any("CtxDeviceKey")

func ContextDevice(ctx context.Context) string {
	value, ok := ctx.Value(CtxDeviceKey).(string)
	if ok {
		return value
	}

	return ""
}

func ContextWithDevice(ctx context.Context, deviceid string) context.Context {
	return context.WithValue(ctx, CtxDeviceKey, deviceid)
}
