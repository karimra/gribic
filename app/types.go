package app

import "time"

const (
	defaultGrpcPort   = "57401"
	msgSize           = 512 * 1024 * 1024
	defaultRetryTimer = 10 * time.Second
)

type TargetError struct {
	TargetName string
	Err        error
}
