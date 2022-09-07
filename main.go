package main

import (
	"fmt"
	"log"
	"time"
	delbuf "tjweldon/beatbox/src/delay_buffers"
	"tjweldon/beatbox/src/examples"
	"tjweldon/beatbox/src/util"

	"github.com/alexflint/go-arg"
	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
)

// set up the main logger
var logger = util.Logger{}.Ctx("main.go").Vol(util.Loud)

// set the global log volume
var _ = util.LogVolume.FilterBelow(util.Quiet)

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

// Init is a wrapper for arg.MustParse
func (c Cli) Init() Cli {
	arg.MustParse(&c)
	return c
}

var args = Cli{}.Init()

func main() {
	logger := logger.Ctx("main").Vol(util.Loud)

	// create the audio stream
	stream := examples.DrumMachine(Kick, Clap, Hat, args.Loop, Format, Tempo)
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
