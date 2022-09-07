package streams

// Sequencer is a Streamer that is initialised with a sequence of booleans
// that represent whether a sound should be played or not
type Sequencer struct {
	Seq   []bool
	Loop  bool
	Sound Stream
}

// Stream implements the Generator interface for Mixer
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

		logger.Log("sequence with sound", seq.Sound, ": step", n, "is", seq.Seq[n])

		// otherwise send either the sound
		if seq.Seq[n] {
			s = seq.Sound()
		} else {
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
