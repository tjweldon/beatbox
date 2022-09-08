package synth

import (
	"log"
	"math"
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
    curve func(sampleIdx int) (volume float64)
    base float64
}

func (vc *volumeCurve) Stream(samples [][2]float64) (n int, ok bool) {
    n, ok = vc.Streamer.Stream(samples)

    gain := func(sIdx int) float64 {
        return math.Pow(vc.base, vc.curve(sIdx))
    }

    for i := range samples[:n] {
        samples[i][0] *= gain(i)
        samples[i][1] *= gain(i)
    }

    return n, ok
}

func (vc *volumeCurve) Err() error { return vc.Streamer.Err() }


type Envelope struct {
    Attack, Decay, Sustain, Release time.Duration
    curves [4]func(sampleIdx int) float64
    incoming streams.Stream
    format beep.Format
}

func (e Envelope)Init(atk, stn, dcy, rel time.Duration, format beep.Format) *Envelope {
    e = Envelope{Attack: atk, Sustain: stn, Decay: dcy, Release: rel, format: format}
    
    sampleAtk := format.SampleRate.N(e.Attack)
    atkCurve := func(sampleIdx int) float64 {
        if sampleIdx >= sampleAtk {
            return 1
        } else if sampleIdx < 0 {
            return 0
        }
        return float64(sampleIdx)/float64(sampleAtk)
    }

    stnOffset := sampleAtk
    sampleStn := format.SampleRate.N(stn)
    sustainCurve := func(sampleIdx int) float64 {
        return 1
    }

    dcyOffset := stnOffset + sampleStn
    sampleDcy := format.SampleRate.N(dcy)
    decayCurve := func(sampleIdx int) float64 {
        relIdx := sampleIdx - dcyOffset
        if relIdx < 0 {
            return 1
        }
        relIdx = Min(sampleDcy, relIdx)

        return math.Pow(math.E, )
    }
}

func (e Envelope) Curve(sampleIdx int) float64 {
    sampleEnv := util.Map(
        e.format.SampleRate.N,
        []time.Duration{e.Attack, e.Sustain, e.Decay, e.Release},
    )
    switch {
    case 0 <= sampleIdx && sampleIdx < sampleEnv[0]:
    }
}

func (e *Envelope) Stream() streams.Stream {
    outStream := func() *streams.FStreamer {
        inc := e.incoming()
        return streams.F(inc.Format, &volumeCurve{
            Streamer: inc, 
            curve: func(sampleIdx int) (volume float64) {return 1}, base: 2,
        })
    }

    return outStream
}

func Oscillator(format beep.Format, freq int) streams.Stream {
    return func () *streams.FStreamer {
        osc, err := generators.SinTone(format.SampleRate, freq)
        if err != nil {
            log.Fatal(err)
        }
        return streams.F(format, osc)
    }
}
