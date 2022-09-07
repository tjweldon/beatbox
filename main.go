package main

import (
	"fmt"
	"log"
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

type Cli struct {
	Loop bool `arg:"-l,--loop" help:"loop the sequence stream" default:"false"`
}

func (c Cli) Init() Cli {
	arg.MustParse(&c)
	return c
}

var args = Cli{}.Init()

func main() {
	logger := logger.Ctx("main")

	sequencer2Stream := func(s streams.Sequencer) streams.Stream { return s.Stream() }

	// composition if instruments and their sequences
	instruments := util.Map(
		sequencer2Stream,
		[]streams.Sequencer{
			// 4 to the floor kick drum
			{
				Seq:   []bool{true, false},
				Loop:  args.Loop,
				Sound: streams.MakeStreamBuf(Kick).Stream(),
			},

			// hats on 16ths
			{
				Seq:   []bool{true},
				Loop:  args.Loop,
				Sound: streams.MakeStreamBuf(Hat).Stream(),
			},

			// off beat clap
			{
				Seq:   []bool{false, false, true, false},
				Loop:  args.Loop,
				Sound: streams.MakeStreamBuf(Clap).Stream(),
			},
		},
	)

	// an AudioBuf whose stream is fed to the speaker
	mixed := streams.Mixer{
		Tracks: instruments,
		Format: Format,
	}.Stream()

	// quantise the mixed sequences to the tempo & quantisation
	quantised := streams.Quantiser{
		Tempo:        Tempo,
		Quantisation: delbuf.Sixteenth,
		Format:       Format,

		// a mixer feeding into the Quantiser
		Incoming: mixed,
	}.Stream()

	// buffer the output stream
	stream := streams.AudioBuf{
		QuantaCount: 4,
		Tempo:       Tempo,
		Format:      Format,

		// a Quantiser feeding into the AudioBuf
		Incoming: quantised,
	}.Stream()
	logger.Log("built stream")

	// initialising the speaker
	err := speaker.Init(Format.SampleRate, Format.SampleRate.N(time.Second/10))
	if err != nil {
		log.Fatal(err)
	}
	logger.Log("initialised speaker")

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
