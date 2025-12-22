package common

//
// logging.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

const (
	LogKeyUserName = "user_name"
	LogKeyDeviceID = "device_id"
)

const (
	LogKeyAuthResult     = "auth_result"
	LogAuthResultSuccess = "success"
	LogAuthResultFailed  = "failed"
	LogAuthResultError   = "error"
)

const (
	LogKeyReqID           = "req_id"
	LogKeyRequestHeaders  = "req_headers"
	LogKeyResponseHeaders = "resp_headers"
)
