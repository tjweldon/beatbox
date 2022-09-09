package synth

import (
	"log"
	"time"
	"tjweldon/beatbox/src/streams"
	"tjweldon/beatbox/src/util"

	"github.com/faiface/beep"
	"github.com/faiface/beep/generators"
)

// MakeSynth is the entrypoint
func MakeSynth(format beep.Format) streams.Stream {
	buf := beep.NewBuffer(format)

	return streams.MakeStreamBuf(buf).Stream()
}

type volumeCurve struct {
	Streamer beep.Streamer
	curve    func(sampleIdx int) (volume float64)
	base     float64
}

func (vc *volumeCurve) Stream(samples [][2]float64) (n int, ok bool) {
	n, ok = vc.Streamer.Stream(samples)
	gain := vc.curve

	for i := range samples[:n] {
		samples[i][0] *= gain(i)
		samples[i][1] *= gain(i)
	}

	return n, ok
}

func (vc *volumeCurve) Err() error { return vc.Streamer.Err() }

type Envelope struct {
	Gate                   time.Duration
	Attack, Decay, Release time.Duration
	Sustain                float64

	incoming streams.Stream
	format   beep.Format
}

func (*Envelope) Init(atk, dcy, rel time.Duration, stn float64, format beep.Format) *Envelope {
	return &Envelope{Attack: atk, Sustain: stn, Decay: dcy, Release: rel, format: format}
}

func (e *Envelope) SetIncoming(streamer streams.Stream) {
	e.incoming = streamer
}

func (e *Envelope) Hold() time.Duration {
	return e.Gate - e.Attack - e.Decay
}

type point struct {
	n   int
	vol float64
}

func (p point) minus(q point) point {
	return point{p.n - q.n, p.vol - q.vol}
}

func (p point) lineTo(q point) func(n int) float64 {
	return func(n int) float64 {
		deltaVol := q.vol - p.vol
		deltaIdx := float64(q.n - p.n)
		return deltaVol*float64(n-p.n)/deltaIdx + p.vol
	}
}

func (e *Envelope) Curve(sampleIdx int) float64 {
	sampleEnv := util.Map(
		e.format.SampleRate.N,
		[]time.Duration{e.Attack, e.Decay, e.Hold(), e.Release},
	)
	var starts []int
	total := 0
	for _, v := range sampleEnv {
		starts = append(starts, total)
		total += v
	}

	// envelope model:
	//
	//            <------------------- gate ------------------->
	//            <--- attack ---><--- decay ---><--- hold ---><--- release --->
	//	     0db |             * |*             |              |               |
	//	     ^   |           *   |   *          |              |               |
	//	     |   |         *     |       *      |              |               |
	//	  volume |       *       |          *   |              |               |
	//	     |   |     *         |             *|   *   *   *  | *             |
	// 	     |   |   *           |              |        T     |      *        |
	//	         | *             |              |     sustain  |          *    |
	//    -100db *               |              |        |     |               *
	//	         0ms-----------------------------------------------------------500ms
	//             -- time ->
	//
	//
	switch {

	// attack: -100db -> 0db
	case starts[0] <= sampleIdx && sampleIdx < sampleEnv[0]:
		start, end := point{starts[0], 0.0}, point{sampleEnv[0], 1.0}
		return start.lineTo(end)(sampleIdx)

	// decay: 0db -> sustain
	case starts[1] <= sampleIdx && sampleIdx < sampleEnv[1]:
		start, end := point{starts[1], 1.0}, point{sampleEnv[1], e.Sustain}
		return start.lineTo(end)(sampleIdx)

	// hold: sustain -> sustain
	case starts[2] <= sampleIdx && sampleIdx < sampleEnv[2]:
		start, end := point{starts[2], e.Sustain}, point{sampleEnv[2], e.Sustain}
		return start.lineTo(end)(sampleIdx)

	// release: sustain -> -100db
	case starts[3] <= sampleIdx && sampleIdx < sampleEnv[3]:
		start, end := point{starts[3], e.Sustain}, point{sampleEnv[3], 0.0}
		return start.lineTo(end)(sampleIdx)

	// or silence: -100db -> -100db
	default:
		return point{vol: -100.0}.lineTo(point{n: sampleIdx, vol: -100.0})(sampleIdx)
	}
}

func (e *Envelope) Stream() streams.Stream {
	outStream := func() *streams.FStreamer {
		inc := e.incoming()
		return streams.F(
			inc.Format,
			&volumeCurve{
				Streamer: inc,
				curve:    e.Curve,
				base:     10, // Decibels
			},
		)
	}

	return outStream
}

func Oscillator(format beep.Format, freq int) streams.Stream {
	return func() *streams.FStreamer {
		osc, err := generators.SinTone(format.SampleRate, freq)
		if err != nil {
			log.Fatal(err)
		}
		return streams.F(format, osc)
	}
}
