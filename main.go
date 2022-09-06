package main

import (
	"fmt"
	"log"
	"time"

	delbuf "tjweldon/beatbox/delay_buffers"
	"tjweldon/beatbox/util"

	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
)

// Logger is a context-aware logger
type Logger struct {
	prefixes []any
}

// Ctx returns a copy of the logger with the given prefix added after all pre-existing prefixes
func (l Logger) Ctx(prefix string) Logger {
	return Logger{append(l.prefixes, prefix+":")}
}

// Log shares its interface with log.Println
func (l Logger) Log(msgs ...any) {
	log.Println(append(l.prefixes, msgs...)...)
}

// set up the main logger
var logger = Logger{}.Ctx("main.go")

// Kick, Clap, Hat are loaded on module load
var (
	Kick = util.BufferSample("./kick.wav")
	Clap = util.BufferSample("./clap.wav")
	Hat  = util.BufferSample("./hat.wav")
	// Empty is literally an empty buffer
	Empty = beep.NewBuffer(Kick.Format())
)

// Tempo is the tempo of the song in beats per minute
var Tempo = delbuf.Tempo(128)

// Format is the format of the samples
var Format = Kick.Format()

// Beat is a Stream that always returns a streamer that is at least one beat long and silent
var Beat = func() beep.Streamer { return beep.Silence(Tempo.Count(Format)) }

// Stream is a synchronous generator of beep.Streamers
type Stream func() beep.Streamer

// MakeStream returns a Stream that always plays the buffered sample
func MakeStream(buf *beep.Buffer) Stream {
	stream := func() beep.Streamer {
		return buf.Streamer(0, buf.Len())
	}

	return stream
}

// Sequencer is a Streamer that is initialised with a sequence of booleans
// that represent whether a sound should be played or not
type Sequencer func(stream Stream, seq []bool, loop bool) Stream

// Sequence is a Sequencer implementation
func Sequence(stream Stream, seq []bool, loop bool) Stream {
	logger := logger.Ctx("Sequence")
	n := 0
	emptyGen := MakeStream(Empty)

	logger.Log("initialising sequence outStream")
	outStream := func() beep.Streamer {
		logger := logger.Ctx("outStream")
		var s beep.Streamer
		// if we're done with the sequence, return nil
		if n >= len(seq) {
			logger.Log("sequence complete")
			return nil
		}

		// otherwise send either the sound
		if seq[n] {
			s = stream()
		} else {
			s = emptyGen()
		}

		n = (n + 1)
		if loop {
			n = n % len(seq)
		}

		return s
	}

	return outStream
}

// Mixer takes a slice of Stream functions and returns a Stream function which
// is the superposition of all the streams.
type Mixer func([]Stream) Stream

// Mix is an implementation of Mixer
func Mix(streams []Stream) Stream {
	logger := logger.Ctx("Mix")
	logger.Log("initialising mix outStream")
	outStream := func() beep.Streamer {
		logger.Ctx("outStream")
		buf := beep.NewBuffer(Format)
		sequenceStep := make([]beep.Streamer, len(streams))
		allClosed := false
		for i, incoming := range streams {
			sequenceStep[i] = incoming()

			// if sequenceStep[i] == nil is true for all incoming
			// then we're done
			allClosed = allClosed && sequenceStep[i] == nil
		}

		// allClosed being true indicates we're done
		if allClosed {
			logger.Log("all sequences exhausted")
			return nil
		}
		buf.Append(beep.Mix(sequenceStep...))

		logger.Log("created buffer of", buf.Len(), "samples")
		return buf.Streamer(0, buf.Len())
	}

	return outStream
}

// Quantiser is a function that takes a Stream, a tempo and a quantisation and
// returns a Stream that is quantised to the given tempo and quantisation
type Quantiser func(Stream, timing delbuf.Tempo, q delbuf.Quantisation) Stream

// Quantise is an implementation of Quantiser
func Quantise(stream Stream, tempo delbuf.Tempo, q delbuf.Quantisation) Stream {
	logger := logger.Ctx("Quantise")
	buf := beep.NewBuffer(Format)
	timing := delbuf.Timing{}.From(Tempo, Format).Quantise(q)

	logger.Log("initialising")
	outStream := func() beep.Streamer {
		logger := logger.Ctx("outStream")
		truncated := delbuf.TruncateHead(buf, timing.Samples)
		buf = beep.NewBuffer(Format)
		nxt := stream()

		// handle upstream exhausted
		if nxt == nil {
			logger.Log("upstream exhausted")
			return nil
		}

		// Create a buffer made of...
		buf.Append(
			// a mix of the following streamers:
			beep.Mix(
				// Beat makes sure the quantised chunk is at least one beat long
				Beat(),
				// This is the new sounds coming in from stream
				nxt,
				// The tail end of the previous chunk
				truncated.Streamer(0, truncated.Len()),
			),
		)

		// send one beat's worth
		logger.Log("sending a quantum of tunage")
		return buf.Streamer(0, timing.Samples)
	}

	return outStream
}

// PopBuffer pops a streamer from the head of a buffer and returns both the head as a streamer and
// the truncated tail as a buffer
func PopBuffer(buf *beep.Buffer, upto int) (head beep.Streamer, tail *beep.Buffer) {
	logger := logger.Ctx("PopBuffer")
	head = buf.Streamer(0, upto)
	tail = delbuf.TruncateHead(buf, upto-1)

	logger.Log("popped", upto, " off head of buffer with length", buf.Len(), ";", tail.Len(), "remaining")
	return head, tail
}

// AudioBuffer is a function that takes a Stream and uses it to populate a
// buffer that is used to store a set amound of pre-rendered audio. The
// generator should send the buffer's contents to the speaker and then refill
type AudioBuf func(stream Stream, lengthQuanta int) Stream

// BufferAudio is an implementation of AudioBuf
func BufferAudio(stream Stream, lengthQuanta int, timing delbuf.Timing) Stream {
	logger := logger.Ctx("BufferAudio")
	logger.Log("initialising buffer outStream")

	buf := beep.NewBuffer(Format)
	outStream := func() beep.Streamer {
		logger.Ctx("outStream")
		for buf.Len() <= timing.Samples*lengthQuanta {
			nxt := stream()
			if nxt == nil {
				return nil
			}
			buf.Append(nxt)
		}
		var out beep.Streamer
		logger.Log("buffer has", buf.Len(), "samples")
		out, buf = PopBuffer(buf, timing.Samples)
		return out
	}

	return outStream
}

func main() {
	logger := logger.Ctx("main")
	stream := BufferAudio(
		Quantise(
			Mix(
				[]Stream{
					Sequence(MakeStream(Kick), []bool{true, false}, true),
					Sequence(MakeStream(Hat), []bool{true}, true),
					Sequence(MakeStream(Clap), []bool{false, false, true, false}, true),
				},
			),
			Tempo,
			delbuf.Sixteenth,
		),
		4,
		delbuf.Timing{}.From(Tempo, Format),
	)
	logger.Log("built stream")

	speaker.Init(Format.SampleRate, Format.SampleRate.N(time.Second/10))
	logger.Log("initialising speaker")

	speaker.Play(beep.Iterate(stream))
	logger.Log("playing")

	Timer(time.Now())
}

// Timer prints the current time every half second. It also blocks the thread forever
// so that the audio can play
func Timer(start time.Time) {
	clock := time.NewTicker(time.Second / 2)
	for tick := range clock.C {
		fmt.Printf("\rTime: % 20s", tick.Sub(start))
	}
}
