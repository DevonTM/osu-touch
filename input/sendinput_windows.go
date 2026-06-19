//go:build windows

package input

import (
	"errors"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	vkZ = 0x5A
	vkX = 0x58

	inputKeyboard = 1
	keyEventKeyUp = 0x0002
)

var procSendInput = windows.NewLazySystemDLL("user32.dll").NewProc("SendInput")

// input mirrors Win32 INPUT on 64-bit Windows. The union must be large enough
// for MOUSEINPUT too, so cbSize is 40 bytes even when sending KEYBDINPUT.
type input struct {
	Type uint32
	_    uint32
	Ki   keyboardInput
	_    [8]byte
}

type keyboardInput struct {
	Vk        uint16
	Scan      uint16
	Flags     uint32
	Time      uint32
	ExtraInfo uintptr
}

func ReleaseAll() error {
	var errs []error
	if err := releaseKey(vkZ); err != nil {
		errs = append(errs, err)
	}
	if err := releaseKey(vkX); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func ApplyMask(oldMask, newMask byte) error {
	oldMask &= 0x03
	newMask &= 0x03
	if oldMask == newMask {
		return nil
	}

	if err := applyKeyBit(oldMask, newMask, 0x01, vkZ); err != nil {
		return err
	}
	return applyKeyBit(oldMask, newMask, 0x02, vkX)
}

func applyKeyBit(oldMask, newMask, bit byte, vk uint16) error {
	wasDown := oldMask&bit != 0
	isDown := newMask&bit != 0
	if wasDown == isDown {
		return nil
	}
	if isDown {
		return pressKey(vk)
	}
	return releaseKey(vk)
}

func pressKey(vk uint16) error {
	return sendKey(vk, 0)
}

func releaseKey(vk uint16) error {
	return sendKey(vk, keyEventKeyUp)
}

func sendKey(vk uint16, flags uint32) error {
	inputs := []input{{
		Type: inputKeyboard,
		Ki: keyboardInput{
			Vk:    vk,
			Flags: flags,
		},
	}}
	return sendInputs(inputs)
}

func sendInputs(inputs []input) error {
	if len(inputs) == 0 {
		return nil
	}

	r1, _, err := procSendInput.Call(
		uintptr(len(inputs)),
		uintptr(unsafe.Pointer(&inputs[0])),
		uintptr(int32(unsafe.Sizeof(input{}))),
	)
	if r1 != uintptr(len(inputs)) {
		if err != windows.ERROR_SUCCESS {
			return fmt.Errorf("SendInput: %w", err)
		}
		return fmt.Errorf("SendInput: sent %d of %d inputs", r1, len(inputs))
	}
	return nil
}
