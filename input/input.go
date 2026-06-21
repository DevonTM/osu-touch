package input

type Key struct {
	Label string
	VK    uint16
	Note  uint8
}

type Keys struct {
	First        Key
	Second       Key
	MIDIChannel  int
	MIDIVelocity int
	MIDIPortName string
}
