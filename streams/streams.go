package streams

import (
	delbuf "tjweldon/beatbox/delay_buffers"
	"tjweldon/beatbox/util"

	"github.com/faiface/beep"
)

// Empty is literally an empty buffer
var Empty = beep.NewBuffer(beep.Format{SampleRate: 44100, NumChannels: 2, Precision: 3})

var logger = util.Logger{}.Ctx("streams")

// Stream is a synchronous generator of beep.Streamers
type Stream func() *FStreamer

// Gen is a convenience function for creating a generator from a Stream
func (s Stream) Gen() func() beep.Streamer {
	return func() beep.Streamer {
		return s()
	}
}

// FStreamer is a Streamer that also knows its Format
type FStreamer struct {
	beep.Streamer
	Format beep.Format
}

// F is a convenience function for creating FStreamers
func F(f beep.Format, s beep.Streamer) *FStreamer {
	return &FStreamer{s, f}
}

// Generator is a struct that transforms the generated sound somehow
type Generator interface {
	Stream() Stream
}

// StreamBuf returns a Stream that always plays the buffered sample
type StreamBuf struct {
	buf *beep.Buffer
}

func MakeStreamBuf(buf *beep.Buffer) StreamBuf {
	return StreamBuf{buf: buf}
}

func (sb StreamBuf) All() beep.Streamer {
	return sb.buf.Streamer(0, sb.buf.Len())
}

func (sb StreamBuf) Stream() Stream {
	stream := func() *FStreamer {
		return F(sb.buf.Format(), sb.All())
	}

	return stream
}

// Sequencer is a Streamer that is initialised with a sequence of booleans
// that represent whether a sound should be played or not
type Sequencer struct {
	Seq   []bool
	Loop  bool
	Sound Stream
}

// Sequence is a Sequencer implementation
func (seq Sequencer) Stream() Stream {
	logger := logger.Ctx("Sequence")
	n := 0
	emptyGen := StreamBuf{Empty}.Stream()

	logger.Log("initialising sequence outStream")
	outStream := func() *FStreamer {
		logger := logger.Ctx("outStream")
		var s *FStreamer
		// if we're done with the sequence, return nil
		if n >= len(seq.Seq) {
			logger.Log("sequence complete")
			return nil
		}

		// otherwise send either the sound
		if seq.Seq[n] {
			s = seq.Sound()
		} else {
			s = emptyGen()
		}

		// handle streamer exhausted
		if s == nil {
			return nil
		}

		n = (n + 1)
		if seq.Loop {
			n = n % len(seq.Seq)
		}

		return s
	}

	return outStream
}

// Mixer takes a slice of Stream functions and returns a Stream function which
// is the superposition of all the streasb.
type Mixer struct {
	Tracks []Stream
	Format beep.Format
}

// Mix is an implementation of Mixer
func (m Mixer) Stream() Stream {
	logger := logger.Ctx("Mix")
	logger.Log("initialising mix outStream")
	outStream := func() *FStreamer {
		logger.Ctx("outStream")

		sequenceStep := make([]beep.Streamer, len(m.Tracks))
		allClosed := false
		for i, incoming := range m.Tracks {
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
		buf := beep.NewBuffer(m.Format)
		buf.Append(beep.Mix(sequenceStep...))

		logger.Log("created buffer of", buf.Len(), "samples")
		return F(buf.Format(), buf.Streamer(0, buf.Len()))
	}

	return outStream
}

// Quantiser is a function that takes a Stream, a Tempo and a quantisation and
// returns a Stream that is quantised to the given Tempo and quantisation
type Quantiser struct {
	Incoming     Stream
	Tempo        delbuf.Tempo
	Quantisation delbuf.Quantisation
	Format       beep.Format
}

// Quantise is an implementation of Quantiser
func (q Quantiser) Stream() Stream {
	logger := logger.Ctx("Quantise")
	buf := beep.NewBuffer(q.Format)
	timing := delbuf.Timing{}.From(q.Tempo, q.Format).Quantise(q.Quantisation)

	logger.Log("initialising")
	outStream := func() *FStreamer {
		logger := logger.Ctx("outStream")
		truncated := delbuf.TruncateHead(buf, timing.Samples)
		buf = beep.NewBuffer(q.Format)
		nxt := q.Incoming()

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
				beep.Silence(timing.Samples),
				// This is the new sounds coming in from Incoming
				nxt,
				// The tail end of the previous chunk
				truncated.Streamer(0, truncated.Len()),
			),
		)

		// send one beat'Tracks worth
		logger.Log("sending a quantum of tunage")
		return F(q.Format, buf.Streamer(0, timing.Samples))
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
// generator should send the buffer'Tracks contents to the speaker and then refill
type AudioBuf struct {
	Incoming    Stream
	QuantaCount int
	Tempo       delbuf.Tempo
	Format      beep.Format
}

// BufferAudio is an implementation of AudioBuf
func (ab AudioBuf) Stream() Stream {

	logger := logger.Ctx("BufferAudio")
	logger.Log("initialising buffer outStream")

	buf := beep.NewBuffer(ab.Format)
	timing := delbuf.Timing{}.From(ab.Tempo, ab.Format)

	// set up the outgoing Incoming
	outStream := func() *FStreamer {
		logger.Ctx("outStream")

		// the incoming audio gets appended to the audio Incoming here
		// until the buffer contains enough audio
		for buf.Len() <= timing.Samples*ab.QuantaCount {
			nxt := ab.Incoming()
			if nxt == nil {
				return nil
			}
			buf.Append(nxt)
		}
		var out beep.Streamer
		logger.Log("buffer has", buf.Len(), "samples")

		// pop the head off the buffer
		out, buf = PopBuffer(buf, timing.Samples)
		return F(buf.Format(), out)
	}

	return outStream
}

func MixAll[T Generator](format beep.Format, gens []T) Generator {
	streamers := []Stream{}
	for _, gen := range gens {
		streamers = append(streamers, gen.Stream())
	}

	return Mixer{Tracks: streamers, Format: format}
}
