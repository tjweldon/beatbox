package delay_buffers

import (
	"github.com/faiface/beep"
	"time"
)

// DelaySound takes a sample and a delay as a time.Duration and returns a streamer that
// streams the sample after silence for the delay specified.
// This function allows us to treat every sample play as and independent sound
// and layer everything together
func DelaySound(by time.Duration, sound *beep.Buffer) (beep.Streamer, beep.Format) {
	// create a buffer to hold the result
	delayBuffer := beep.NewBuffer(sound.Format())

	// silence for the duration passed
	delaySilence := beep.Silence(sound.Format().SampleRate.N(by))

	// append silence to the empty buffer
	delayBuffer.Append(delaySilence)

	// the sample to be played
	delayBuffer.Append(sound.Streamer(0, sound.Len()))

	return delayBuffer.Streamer(0, delayBuffer.Len()), delayBuffer.Format()
}

// TruncateHead truncates the head of the buffer to the specified number of samples
func TruncateHead(buf *beep.Buffer, samples int) *beep.Buffer {
	truncated := beep.NewBuffer(buf.Format())
	if buf.Len() < samples {
		return truncated
	}
	truncated.Append(buf.Streamer(samples, buf.Len()))
	return truncated
}

// Add creates a buffer that is the superposition of the two buffers passed.
func Add(buffer *beep.Buffer, streamer beep.Streamer) *beep.Buffer {
	buf := beep.NewBuffer(buffer.Format())
	buf.Append(beep.Mix(buffer.Streamer(0, buffer.Len())))

	return buf
}

// Tempo is a type that represents a tempo in beats per minute
type Tempo int

// Quantum returns the duration of a single beat
func (t Tempo) Quantum() time.Duration {
	return time.Minute / time.Duration(t)
}

// Count returns the number of samples in a single beat for a given format.
func (t Tempo) Count(of beep.Format) (samples int) {
	return of.SampleRate.N(t.Quantum())
}

type Timing struct {
	Duration time.Duration
	Samples  int
}

func (Timing) From(t Tempo, f beep.Format) Timing {
	return Timing{Duration: t.Quantum(), Samples: t.Count(f)}
}

func (t Timing) Quantise(q Quantisation) Timing {
	result := Timing{
		Samples:  t.Samples / int(q),
		Duration: t.Duration / time.Duration(q),
	}
	return result
}

type Quantisation int

const (
	Quarter Quantisation = 0<<1 + iota
	Eighth
	Sixteenth
)

// PopBuffer pops a streamer from the head of a buffer and returns both the head as a streamer and
// the truncated tail as a buffer
func PopBuffer(buf *beep.Buffer, upto int) (head beep.Streamer, tail *beep.Buffer) {
	head = buf.Streamer(0, upto)
	tail = TruncateHead(buf, upto-1)

	return head, tail
}
