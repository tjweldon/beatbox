package streams

import (
	"github.com/faiface/beep"
	"tjweldon/beatbox/src/util"
)

// Empty is literally an empty buffer
var Empty = beep.NewBuffer(beep.Format{SampleRate: 44100, NumChannels: 2, Precision: 3})

var logger = util.Logger{}.Ctx("streams")

// Stream is a synchronous generator of beep.Streamers
type Stream func() *FStreamer

// Gen is a convenience function for creating a beep.Streamer generator from a Stream
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

// MakeStreamBuf returns a StreamBuf that plays the given sample
func MakeStreamBuf(buf *beep.Buffer) StreamBuf {
	return StreamBuf{buf: buf}
}

// All returns a beep.Streamer that plays the whole buffer
func (sb StreamBuf) All() beep.Streamer {
	return sb.buf.Streamer(0, sb.buf.Len())
}

// Stream implements the Generator interface for Delay
func (sb StreamBuf) Stream() Stream {
	stream := func() *FStreamer {
		return F(sb.buf.Format(), sb.All())
	}

	return stream
}
