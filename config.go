package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/DevonTM/osu-touch/input"
)

const (
	defaultAddr = ":51155"
	addrEnv     = "OSU_TOUCH_ADDR"

	defaultKeys = "Z,X"
	keysEnv     = "OSU_TOUCH_KEYS"

	defaultMIDINotes    = "C4,D4"
	defaultMIDIChannel  = 1
	defaultMIDIVelocity = 127
	defaultMIDIPortName = "osu-touch MIDI"

	midiNotesEnv    = "OSU_TOUCH_MIDI_NOTES"
	midiChannelEnv  = "OSU_TOUCH_MIDI_CHANNEL"
	midiVelocityEnv = "OSU_TOUCH_MIDI_VELOCITY"
	midiPortNameEnv = "OSU_TOUCH_MIDI_PORT_NAME"
)

func serverAddr() string {
	if addr := os.Getenv(addrEnv); addr != "" {
		return addr
	}
	return defaultAddr
}

func inputKeys() input.Keys {
	if runtime.GOOS == "linux" {
		return midiInputKeys()
	}
	return keyboardInputKeys()
}

func keyboardInputKeys() input.Keys {
	value := os.Getenv(keysEnv)
	keys, ok := parseInputKeys(value)
	if ok {
		return keys
	}

	if strings.TrimSpace(value) != "" {
		log.Printf("Warning: Invalid %s=%q; using default %s", keysEnv, value, defaultKeys)
	}
	keys, _ = parseInputKeys(defaultKeys)
	return keys
}

func midiInputKeys() input.Keys {
	value := os.Getenv(midiNotesEnv)
	notes, ok := parseMIDINotes(value)
	if !ok {
		if strings.TrimSpace(value) != "" {
			log.Printf("Warning: Invalid %s=%q; using default %s", midiNotesEnv, value, defaultMIDINotes)
		}
		notes, _ = parseMIDINotes(defaultMIDINotes)
	}

	portName := strings.TrimSpace(os.Getenv(midiPortNameEnv))
	if portName == "" {
		portName = defaultMIDIPortName
	}

	return input.Keys{
		First:        input.Key{Label: midiNoteName(notes[0]), Note: notes[0]},
		Second:       input.Key{Label: midiNoteName(notes[1]), Note: notes[1]},
		MIDIChannel:  parseMIDIIntEnv(midiChannelEnv, defaultMIDIChannel, 1, 16),
		MIDIVelocity: parseMIDIIntEnv(midiVelocityEnv, defaultMIDIVelocity, 1, 127),
		MIDIPortName: portName,
	}
}
func parseInputKeys(value string) (input.Keys, bool) {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" {
		return input.Keys{}, false
	}

	parts := strings.Split(value, ",")
	if len(parts) != 2 {
		return input.Keys{}, false
	}
	first := strings.TrimSpace(parts[0])
	second := strings.TrimSpace(parts[1])
	if len(first) != 1 || len(second) != 1 {
		return input.Keys{}, false
	}
	firstKey := first[0]
	secondKey := second[0]
	if firstKey == secondKey || !isSafeKey(firstKey) || !isSafeKey(secondKey) {
		return input.Keys{}, false
	}
	return input.Keys{
		First:  input.Key{Label: string(firstKey), VK: uint16(firstKey)},
		Second: input.Key{Label: string(secondKey), VK: uint16(secondKey)},
	}, true
}

func isSafeKey(key byte) bool {
	return key >= 'A' && key <= 'Z' || key >= '0' && key <= '9'
}

func parseMIDINotes(value string) ([2]uint8, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return [2]uint8{}, false
	}

	parts := strings.Split(value, ",")
	if len(parts) != 2 {
		return [2]uint8{}, false
	}

	first, ok := parseMIDINote(parts[0])
	if !ok {
		return [2]uint8{}, false
	}
	second, ok := parseMIDINote(parts[1])
	if !ok || first == second {
		return [2]uint8{}, false
	}
	return [2]uint8{first, second}, true
}

func parseMIDINote(value string) (uint8, bool) {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" {
		return 0, false
	}
	if parsed, err := strconv.Atoi(value); err == nil {
		if parsed < 0 || parsed > 127 {
			return 0, false
		}
		return uint8(parsed), true
	}

	if !isMIDINoteLetter(value[0]) {
		return 0, false
	}
	semitone := map[byte]int{
		'C': 0,
		'D': 2,
		'E': 4,
		'F': 5,
		'G': 7,
		'A': 9,
		'B': 11,
	}[value[0]]

	octaveStart := 1
	if octaveStart < len(value) {
		switch value[octaveStart] {
		case '#':
			semitone++
			octaveStart++
		case 'B':
			semitone--
			octaveStart++
		}
	}
	if semitone < 0 {
		semitone += 12
	} else if semitone > 11 {
		semitone -= 12
	}

	octave, err := strconv.Atoi(value[octaveStart:])
	if err != nil {
		return 0, false
	}
	note := (octave+1)*12 + semitone
	if note < 0 || note > 127 {
		return 0, false
	}
	return uint8(note), true
}

func isMIDINoteLetter(value byte) bool {
	return value >= 'A' && value <= 'G'
}

func midiNoteName(note uint8) string {
	names := [...]string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}
	return fmt.Sprintf("%s%d", names[note%12], int(note)/12-1)
}

func parseMIDIIntEnv(name string, defaultValue, minValue, maxValue int) int {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < minValue || parsed > maxValue {
		log.Printf("Warning: Invalid %s=%q; using default %d", name, value, defaultValue)
		return defaultValue
	}
	return parsed
}
