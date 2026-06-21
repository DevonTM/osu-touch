//go:build linux

package input

/*
#cgo LDFLAGS: -lasound
#include <stdlib.h>
#include <alsa/asoundlib.h>

static int osu_touch_open_seq(snd_seq_t **seq) {
	return snd_seq_open(seq, "default", SND_SEQ_OPEN_OUTPUT, 0);
}

static int osu_touch_create_port(snd_seq_t *seq, const char *name) {
	return snd_seq_create_simple_port(seq, name,
		SND_SEQ_PORT_CAP_READ | SND_SEQ_PORT_CAP_SUBS_READ,
		SND_SEQ_PORT_TYPE_MIDI_GENERIC | SND_SEQ_PORT_TYPE_APPLICATION);
}

static int osu_touch_note_on(snd_seq_t *seq, int port, int channel, int note, int velocity) {
	snd_seq_event_t ev;
	snd_seq_ev_clear(&ev);
	snd_seq_ev_set_source(&ev, port);
	snd_seq_ev_set_subs(&ev);
	snd_seq_ev_set_direct(&ev);
	snd_seq_ev_set_noteon(&ev, channel, note, velocity);
	return snd_seq_event_output_direct(seq, &ev);
}

static int osu_touch_note_off(snd_seq_t *seq, int port, int channel, int note) {
	snd_seq_event_t ev;
	snd_seq_ev_clear(&ev);
	snd_seq_ev_set_source(&ev, port);
	snd_seq_ev_set_subs(&ev);
	snd_seq_ev_set_direct(&ev);
	snd_seq_ev_set_noteoff(&ev, channel, note, 0);
	return snd_seq_event_output_direct(seq, &ev);
}

static int osu_touch_all_notes_off(snd_seq_t *seq, int port, int channel) {
	snd_seq_event_t ev;
	snd_seq_ev_clear(&ev);
	snd_seq_ev_set_source(&ev, port);
	snd_seq_ev_set_subs(&ev);
	snd_seq_ev_set_direct(&ev);
	snd_seq_ev_set_controller(&ev, channel, 123, 0);
	return snd_seq_event_output_direct(seq, &ev);
}
*/
import "C"

import (
	"errors"
	"fmt"
	"log"
	"unsafe"
)

type Controller struct {
	seq      *C.snd_seq_t
	port     C.int
	keys     Keys
	channel  C.int
	velocity C.int
	closed   bool
}

func NewController(keys Keys) (*Controller, error) {
	var seq *C.snd_seq_t
	if errCode := C.osu_touch_open_seq(&seq); errCode < 0 {
		return nil, alsaError("open ALSA sequencer", errCode)
	}

	clientName := C.CString(keys.MIDIPortName)
	C.snd_seq_set_client_name(seq, clientName)
	C.free(unsafe.Pointer(clientName))

	portName := C.CString(keys.MIDIPortName)
	port := C.osu_touch_create_port(seq, portName)
	C.free(unsafe.Pointer(portName))
	if port < 0 {
		C.snd_seq_close(seq)
		return nil, alsaError("create MIDI port", port)
	}

	log.Printf("MIDI output: %s (channel %d, notes %d / %d)", keys.MIDIPortName, keys.MIDIChannel, keys.First.Note, keys.Second.Note)

	return &Controller{
		seq:      seq,
		port:     port,
		keys:     keys,
		channel:  C.int(keys.MIDIChannel - 1),
		velocity: C.int(keys.MIDIVelocity),
	}, nil
}

func (c *Controller) Keys() Keys {
	return c.keys
}

func (c *Controller) ReleaseAll() error {
	if c.closed {
		return nil
	}
	var errs []error
	if err := c.noteOff(c.keys.First.Note); err != nil {
		errs = append(errs, err)
	}
	if err := c.noteOff(c.keys.Second.Note); err != nil {
		errs = append(errs, err)
	}
	if err := c.allNotesOff(); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func (c *Controller) Close() error {
	if c.closed {
		return nil
	}
	c.closed = true
	if c.seq != nil {
		C.snd_seq_close(c.seq)
		c.seq = nil
	}
	return nil
}

func (c *Controller) ApplyMask(oldMask, newMask byte) error {
	oldMask &= 0x03
	newMask &= 0x03
	if oldMask == newMask {
		return nil
	}

	if err := c.applyNoteBit(oldMask, newMask, 0x01, c.keys.First.Note); err != nil {
		return err
	}
	return c.applyNoteBit(oldMask, newMask, 0x02, c.keys.Second.Note)
}

func (c *Controller) applyNoteBit(oldMask, newMask, bit byte, note uint8) error {
	wasDown := oldMask&bit != 0
	isDown := newMask&bit != 0
	if wasDown == isDown {
		return nil
	}
	if isDown {
		return c.noteOn(note)
	}
	return c.noteOff(note)
}

func (c *Controller) noteOn(note uint8) error {
	if c.closed {
		return errors.New("MIDI output is closed")
	}
	if errCode := C.osu_touch_note_on(c.seq, c.port, c.channel, C.int(note), c.velocity); errCode < 0 {
		return alsaError("MIDI note on", errCode)
	}
	return nil
}

func (c *Controller) noteOff(note uint8) error {
	if c.closed {
		return nil
	}
	if errCode := C.osu_touch_note_off(c.seq, c.port, c.channel, C.int(note)); errCode < 0 {
		return alsaError("MIDI note off", errCode)
	}
	return nil
}

func (c *Controller) allNotesOff() error {
	if c.closed {
		return nil
	}
	if errCode := C.osu_touch_all_notes_off(c.seq, c.port, c.channel); errCode < 0 {
		return alsaError("MIDI all notes off", errCode)
	}
	return nil
}

func alsaError(action string, errCode C.int) error {
	return fmt.Errorf("%s: %s", action, C.GoString(C.snd_strerror(errCode)))
}
