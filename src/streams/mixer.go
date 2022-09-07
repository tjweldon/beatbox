package streams

import (
	"github.com/faiface/beep"
)

// Mixer takes a slice of Stream functions and returns a Stream function which
// is the superposition of all the Tracks.
type Mixer struct {
	Tracks []Stream
	Format beep.Format
}

// Stream implements the Generator interface for Mixer
func (m Mixer) Stream() Stream {
	logger := logger.Ctx("Mix")
	logger.Log("initialising mix outStream")

	emptyGen := StreamBuf{Empty}.Stream()
	outStream := func() *FStreamer {
		logger.Ctx("outStream")

		sequenceStep := make([]beep.Streamer, len(m.Tracks))
		allClosed := true
		closed := make([]bool, len(m.Tracks))
		for i, incoming := range m.Tracks {
			// if sequenceStep[i] == nil is true for all incoming
			// then we're done
			chunk := incoming()
			closed[i] = chunk == nil

			// allClosed -> false if any chunk is not nil
			allClosed = closed[i] && allClosed

			if chunk == nil {
				// handle upstream exhausted by sending empty buffer
				sequenceStep[i] = emptyGen()
				logger.Log("no sample from upstream track", i, "sending empty buffer")
			} else {
				// otherwise the sound is added to the mix
				sequenceStep[i] = chunk
				logger.Log("got sample from upstream track", i)
			}
		}
		logger.Log("closed:", closed)
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

func MixAll[T Generator](format beep.Format, gens []T) Generator {
	streamers := []Stream{}
	for _, gen := range gens {
		streamers = append(streamers, gen.Stream())
	}

	return Mixer{Tracks: streamers, Format: format}
}
