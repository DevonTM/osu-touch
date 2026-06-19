package main

import "os"

const (
	defaultAddr = ":51155"
	addrEnv     = "OSU_TOUCH_ADDR"
)

func serverAddr() string {
	if addr := os.Getenv(addrEnv); addr != "" {
		return addr
	}
	return defaultAddr
}
