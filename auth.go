package main

import (
	"crypto/rand"
	"crypto/subtle"
	"fmt"
	"math/big"
)

const pinDigits = 6

func newPairingPIN() (string, error) {
	max := big.NewInt(1000000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%0*d", pinDigits, n.Int64()), nil
}

func validPairingPIN(got, want string) bool {
	if len(got) != pinDigits || len(want) != pinDigits {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}
