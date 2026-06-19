package main

import (
	"log"
	"os"
	"strings"

	"github.com/DevonTM/osu-touch/input"
)

const (
	defaultAddr = ":51155"
	addrEnv     = "OSU_TOUCH_ADDR"

	defaultKeys = "ZX"
	keysEnv     = "OSU_TOUCH_KEYS"
)

func serverAddr() string {
	if addr := os.Getenv(addrEnv); addr != "" {
		return addr
	}
	return defaultAddr
}

func inputKeys() input.Keys {
	value := normalizeKeys(os.Getenv(keysEnv))
	keys, ok := parseInputKeys(value)
	if ok {
		return keys
	}

	if value != "" {
		log.Printf("Warning: Invalid %s=%q; using default %s", keysEnv, value, defaultKeys)
	}
	keys, _ = parseInputKeys(defaultKeys)
	return keys
}

func normalizeKeys(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func parseInputKeys(value string) (input.Keys, bool) {
	if len(value) != 2 {
		return input.Keys{}, false
	}
	if value[0] == value[1] || !isSafeKey(value[0]) || !isSafeKey(value[1]) {
		return input.Keys{}, false
	}
	return input.Keys{
		First:  input.Key{Label: string(value[0]), VK: uint16(value[0])},
		Second: input.Key{Label: string(value[1]), VK: uint16(value[1])},
	}, true
}

func isSafeKey(key byte) bool {
	return key >= 'A' && key <= 'Z' || key >= '0' && key <= '9'
}
