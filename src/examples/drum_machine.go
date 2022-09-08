package examples

import (
	"github.com/faiface/beep"
	delbuf "tjweldon/beatbox/src/delay_buffers"
	"tjweldon/beatbox/src/streams"
	"tjweldon/beatbox/src/util"
)

var logger = util.Logger{Volume: util.Loud}.Ctx("examples/drum_machine")

// DrumMachine expects some samples, a loop flag, and audio format and a tempo
// and returns a stream of audio that can be fed to the speaker
func DrumMachine(
	Kick, Clap, Hat *beep.Buffer,
	loop bool,
	format beep.Format,
	tempo delbuf.Tempo,
) streams.Stream {
	logger := logger.Ctx("DrumMachine").Vol(util.Normal)
	// composition if instruments and their sequences
	instruments := util.Map(
		streams.Sequencer.Stream,
		[]streams.Sequencer{
			// 4 to the floor kick drum
			{
				Seq:   []bool{true, false},
				Loop:  loop,
				Sound: streams.MakeStreamBuf(Kick).Stream(),
			},

			// hats on 16ths
			{
				Seq:   []bool{true},
				Loop:  loop,
				Sound: streams.MakeStreamBuf(Hat).Stream(),
			},

			// off beat clap
			{
				Seq:   []bool{false, false, true, false},
				Loop:  loop,
				Sound: streams.MakeStreamBuf(Clap).Stream(),
			},
		},
	)

	logger.Log("instruments")

	// an AudioBuf whose stream is fed to the speaker
	mixed := streams.Mixer{
		Tracks: instruments,
		Format: format,
	}.Stream()

	// quantise the mixed sequences to the tempo & quantisation
	quantised := streams.Quantiser{
		Tempo:        tempo,
		Quantisation: delbuf.Sixteenth,
		Format:       format,

		// a mixer feeding into the Quantiser
		Incoming: mixed,
	}.Stream()

	// buffer the output stream
	return streams.AudioBuf{
		QuantaCount: 4,
		Tempo:       tempo,
		Format:      format,

		// a Quantiser feeding into the AudioBuf
		Incoming: quantised,
	}.Stream()
}
