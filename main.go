package main

import (
	"fmt"
	"time"

	delbuf "tjweldon/beatbox/delay_buffers"
	"tjweldon/beatbox/streams"
	"tjweldon/beatbox/util"

	arg "github.com/alexflint/go-arg"
	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
)

// set up the main logger
var logger = util.Logger{}.Ctx("main.go")

// Kick, Clap, Hat are loaded on module load
var (
	Kick = util.BufferSample("./kick.wav")
	Clap = util.BufferSample("./clap.wav")
	Hat  = util.BufferSample("./hat.wav")
)

// Tempo is the tempo of the song in beats per minute
var Tempo = delbuf.Tempo(128)

// Format is the format of the samples
var Format = Kick.Format()

// Beat is a Stream that always returns a streamer that is at least one beat long and silent
var Beat = func() beep.Streamer { return beep.Silence(Tempo.Count(Format)) }

type Cli struct {
	PrintFormat bool `arg:"-p,--print-format" help:"Prints out the beep format being used globally"`
}

func (c Cli) Init() Cli {
	arg.MustParse(&c)
	return c
}

var args = Cli{}.Init()

func main() {
	logger := logger.Ctx("main")

	if args.PrintFormat {
		fmt.Println(Format)
		fmt.Println(Hat.Format())
		fmt.Println(Clap.Format())
		return
	}

	sequencer2Stream := func(s streams.Sequencer) streams.Stream { return s.Stream() }

	// composition of generators
	instruments := util.Map(
		sequencer2Stream,
		[]streams.Sequencer{
			// 4 to the floor kick drum
			streams.Sequencer{
				Seq:   []bool{true, false},
				Loop:  true,
				Sound: streams.MakeStreamBuf(Kick).Stream(),
			},

			// hats on 16ths
			{
				Seq:   []bool{true},
				Loop:  true,
				Sound: streams.MakeStreamBuf(Hat).Stream(),
			},

			// off beat clap
			{
				Seq:   []bool{false, false, true, false},
				Loop:  true,
				Sound: streams.MakeStreamBuf(Clap).Stream(),
			},
		},
	)

	// create the output stream as:
	// an AudioBuf whose stream is fed to the speaker
	stream := streams.AudioBuf{
		QuantaCount: 4,
		Tempo:       Tempo,
		Format:      Format,

		// a Quantiser feeding into the AudioBuf
		Incoming: streams.Quantiser{
			Tempo:        Tempo,
			Quantisation: delbuf.Sixteenth,
			Format:       Format,

			// a mixer feeding into the Quantiser
			Incoming: streams.Mixer{
				Tracks: instruments,
				Format: Format,
			}.Stream(),
		}.Stream(),
	}.Stream()
	logger.Log("built stream")

	// initialising the speaker
	speaker.Init(Format.SampleRate, Format.SampleRate.N(time.Second/10))
	logger.Log("initialising speaker")

	// playing the stream
	speaker.Play(beep.Iterate(stream.Gen()))
	logger.Log("playing")

	// play and wait for the user to exit
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
