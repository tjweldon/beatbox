package streams

import (
	"fmt"
	"tjweldon/beatbox/src/util"
)

// Sequencer is a Streamer that is initialised with a sequence of booleans
// that represent whether a sound should be played or not
type Sequencer struct {
	Seq   []bool
	Loop  bool
	Sound Stream
}

// Stream implements the Generator interface for Mixer
func (seq Sequencer) Stream() Stream {
	// allocate some state for the generator to enclose
	logger := logger.Ctx("Sequencer.Stream").Vol(util.Normal)
	n := 0
	emptyGen := StreamBuf{Empty}.Stream()

	// set up the generator to be returned
	logger.Log("initialising sequence outStream")
	outStream := func() *FStreamer {
		logger := logger.Ctx("outStream").Vol(util.Quiet)
		var s *FStreamer
		// if we're done with the sequence, return nil
		if n >= len(seq.Seq) {
			logger.Log("sequence complete")
			return nil
		}

		logger.Log("sequence with sound", seq.Sound, ": step", n, "is", seq.Seq[n])

		// otherwise set the next beep.Streamer to be returned
		if seq.Seq[n] {
			// if the signal is true, play the sound
			s = seq.Sound()
		} else {
			// otherwise zero length silence
			s = emptyGen()
		}

		// handle streamer exhausted
		if s == nil {
			logger.Log("instrument exhausted, sending nil")
			return nil
		}

		n++
		if seq.Loop {
			logger.Log("looping sequence")
			n = n % len(seq.Seq)
		}

		logger.Log("sending sequence")
		return s
	}

	return outStream
}

func (seq Sequencer) String() string {
	// Omit the sound from the string representation
	return fmt.Sprintf("Sequencer(%+v)",
		struct {
			Seq  []bool
			Loop bool
		}{seq.Seq, seq.Loop},
	)
}
