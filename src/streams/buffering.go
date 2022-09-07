package streams

import (
	"github.com/faiface/beep"
	"tjweldon/beatbox/src/delay_buffers"
	"tjweldon/beatbox/src/util"
)

// AudioBuf is a function that takes a Stream and uses it to populate a
// buffer that is used to store a set amount of pre-rendered audio. The
// generator should send the buffer's contents to the speaker and then refill
type AudioBuf struct {
	Incoming    Stream
	QuantaCount int
	Tempo       delay_buffers.Tempo
	Format      beep.Format
}

// Stream is an implementation of Generator for AudioBuf
func (ab AudioBuf) Stream() Stream {
	logger := logger.Ctx("BufferAudio")
	logger.Log("initialising buffer outStream")

	buf := beep.NewBuffer(ab.Format)
	timing := delay_buffers.Timing{}.From(ab.Tempo, ab.Format)

	// set up the outgoing Incoming
	outStream := func() *FStreamer {
		logger := logger.Ctx("outStream").Vol(util.Quiet)

		// the incoming audio gets appended to the audio Incoming here
		// until the buffer contains enough audio
		for buf.Len() <= timing.Samples*ab.QuantaCount {
			nxt := ab.Incoming()
			if nxt == nil {
				// play silence if the upstream is exhausted
				buf.Append(beep.Silence(timing.Samples))
				logger.Log("upstream exhausted, sending silence")
			} else {
				buf.Append(nxt)
				logger.Log("got sample from upstream, buffer now", buf.Len(), "samples")
			}
		}
		var out beep.Streamer

		// pop the head off the buffer
		out, buf = delay_buffers.PopBuffer(buf, timing.Samples)
		logger.Log("sending a streamer of length", timing.Samples, "samples")
		logger.Log("buffer of", buf.Len(), "samples remaining")
		return F(buf.Format(), out)
	}

	return outStream
}
