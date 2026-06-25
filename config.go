package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/DevonTM/osu-touch/input"

	"github.com/goccy/go-yaml"
)

const (
	defaultConfigFileName = "config.yaml"
	defaultAddr           = ":51155"
	defaultKeys           = "Z,X"
	defaultMIDINotes      = "C4,D4"
	defaultMIDIChannel    = 1
	defaultMIDIVelocity   = 127
	defaultMIDIPortName   = "osu-touch MIDI"
)

type appConfig struct {
	Addr string     `yaml:"address"`
	Keys string     `yaml:"keys"`
	MIDI midiConfig `yaml:"midi"`
}

type midiConfig struct {
	Notes    string `yaml:"notes"`
	Channel  int    `yaml:"channel"`
	Velocity int    `yaml:"velocity"`
	PortName string `yaml:"port_name"`
}

func defaultAppConfig() appConfig {
	return appConfig{
		Addr: defaultAddr,
		Keys: defaultKeys,
		MIDI: midiConfig{
			Notes:    defaultMIDINotes,
			Channel:  defaultMIDIChannel,
			Velocity: defaultMIDIVelocity,
			PortName: defaultMIDIPortName,
		},
	}
}

func defaultConfigPath() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(exePath), defaultConfigFileName), nil
}

func loadAppConfig(path string) (appConfig, string, error) {
	if strings.TrimSpace(path) == "" {
		defaultPath, err := defaultConfigPath()
		if err != nil {
			return appConfig{}, "", fmt.Errorf("default config path: %w", err)
		}
		path = defaultPath
	}
	path = filepath.Clean(path)

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := writeDefaultConfig(path); err != nil {
			return appConfig{}, path, err
		}
		log.Printf("Created default config: %s", path)
		return defaultAppConfig(), path, nil
	}
	if err != nil {
		return appConfig{}, path, fmt.Errorf("read config %q: %w", path, err)
	}

	config := defaultAppConfig()
	if err := yaml.Unmarshal(data, &config); err != nil {
		return appConfig{}, path, fmt.Errorf("parse config %q: %w", path, err)
	}
	config.normalize()
	return config, path, nil
}

func writeDefaultConfig(path string) error {
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create config directory %q: %w", dir, err)
		}
	}
	data, err := yaml.Marshal(defaultAppConfig())
	if err != nil {
		return fmt.Errorf("encode default config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write default config %q: %w", path, err)
	}
	return nil
}

func (config *appConfig) normalize() {
	config.Addr = strings.TrimSpace(config.Addr)
	if config.Addr == "" {
		config.Addr = defaultAddr
	}

	config.Keys = strings.TrimSpace(config.Keys)
	if config.Keys == "" {
		config.Keys = defaultKeys
	}

	config.MIDI.Notes = strings.TrimSpace(config.MIDI.Notes)
	if config.MIDI.Notes == "" {
		config.MIDI.Notes = defaultMIDINotes
	}

	config.MIDI.Channel = normalizeMIDIInt("midi.channel", config.MIDI.Channel, defaultMIDIChannel, 1, 16)
	config.MIDI.Velocity = normalizeMIDIInt("midi.velocity", config.MIDI.Velocity, defaultMIDIVelocity, 1, 127)

	config.MIDI.PortName = strings.TrimSpace(config.MIDI.PortName)
	if config.MIDI.PortName == "" {
		config.MIDI.PortName = defaultMIDIPortName
	}
}

func (config appConfig) inputKeys() input.Keys {
	if runtime.GOOS == "linux" {
		return config.midiInputKeys()
	}
	return config.keyboardInputKeys()
}

func (config appConfig) keyboardInputKeys() input.Keys {
	keys, ok := parseInputKeys(config.Keys)
	if ok {
		return keys
	}

	if strings.TrimSpace(config.Keys) != "" {
		log.Printf("Warning: Invalid keys=%q; using default %s", config.Keys, defaultKeys)
	}
	keys, _ = parseInputKeys(defaultKeys)
	return keys
}

func (config appConfig) midiInputKeys() input.Keys {
	notes, ok := parseMIDINotes(config.MIDI.Notes)
	if !ok {
		if strings.TrimSpace(config.MIDI.Notes) != "" {
			log.Printf("Warning: Invalid midi.notes=%q; using default %s", config.MIDI.Notes, defaultMIDINotes)
		}
		notes, _ = parseMIDINotes(defaultMIDINotes)
	}

	portName := strings.TrimSpace(config.MIDI.PortName)
	if portName == "" {
		portName = defaultMIDIPortName
	}

	return input.Keys{
		First:        input.Key{Label: midiNoteName(notes[0]), Note: notes[0]},
		Second:       input.Key{Label: midiNoteName(notes[1]), Note: notes[1]},
		MIDIChannel:  config.MIDI.Channel,
		MIDIVelocity: config.MIDI.Velocity,
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

func normalizeMIDIInt(name string, value, defaultValue, minValue, maxValue int) int {
	if value < minValue || value > maxValue {
		log.Printf("Warning: Invalid %s=%d; using default %d", name, value, defaultValue)
		return defaultValue
	}
	return value
}
