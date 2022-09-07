package streams

import (
	"github.com/faiface/beep"
	"tjweldon/beatbox/src/delay_buffers"
	"tjweldon/beatbox/src/util"
)

// Quantiser is a function that takes a Stream, a Tempo and a quantisation and
// returns a Stream that is quantised to the given Tempo and quantisation
type Quantiser struct {
	Incoming     Stream
	Tempo        delay_buffers.Tempo
	Quantisation delay_buffers.Quantisation
	Format       beep.Format
}

// Stream implements the Generator interface for Quantiser
func (q Quantiser) Stream() Stream {
	logger := logger.Ctx("Quantiser.Quantise").Vol(util.Normal)
	buf := beep.NewBuffer(q.Format)
	timing := delay_buffers.Timing{}.From(q.Tempo, q.Format).Quantise(q.Quantisation)

	logger.Log("initialising")
	outStream := func() *FStreamer {
		logger := logger.Ctx("outStream").Vol(util.Quiet)
		truncated := delay_buffers.TruncateHead(buf, timing.Samples)
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

		// send the quantised chunk
		logger.Log("sending quantised chunk.", "Timing:", timing)
		return F(q.Format, buf.Streamer(0, timing.Samples))
	}

	return outStream
}
