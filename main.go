package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/wav"
)

func Min(a, b int) int {
	if a > b {
		return b
	} else {
		return a
	}
}

func Max(a, b int) int {
	if a > b {
		return a
	} else {
		return b
	}
}

type Beat time.Duration

func (b Beat) Tempo() Tempo {
	return Tempo(time.Minute) / Tempo(b)
}
func (b Beat) Sixteenth() Beat {
	return b / 4
}

type Tempo float64

func (t Tempo) BeatDuration() Beat {
	return Beat(time.Minute) / Beat(t)
}
func (t Tempo) Sixteenth() Beat {
	return t.BeatDuration().Sixteenth()
}

type Sample struct {
	path   string
	format beep.Format
	buf    *beep.Buffer
}

func LoadWavSample(path string) (buf *beep.Buffer, format beep.Format, err error) {
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	streamer, format, err := wav.Decode(file)
	if err != nil {
		return
	}
	speaker.Play(streamer)

	buf = beep.NewBuffer(format)
	buf.Append(streamer)
	if err = streamer.Close(); err != nil {
		return
	}

	return buf, format, err
}

func NewSample(path string) (s *Sample, err error) {
	buf, format, err := LoadWavSample(path)
	s = &Sample{path: path, format: format, buf: buf}

	return s, err
}

func (s *Sample) GetFormat() beep.Format     { return s.format }
func (s *Sample) GetDuration() time.Duration { return s.format.SampleRate.D(s.buf.Len()) }
func (s *Sample) GetLen() int                { return s.buf.Len() }

// GetStreamer handles extending the sample with silence if the sustain is longer than the sample
// if the sustain is shorter, the sample is cut off
func (s *Sample) GetStreamer(hold int) beep.StreamSeeker {
	diff := hold - s.buf.Len()
	if diff > 0 {
		s.buf.Append(beep.Silence(diff))
	}
	return s.buf.Streamer(0, hold)
}

type Sequence struct {
	sample   *Sample
	on       [16]bool
	volume   *effects.Volume
	ctrl     *beep.Ctrl
	tempo    Tempo
	streamer beep.Streamer
	err      error
}

func NewSequence(sample *Sample, pattern [16]bool, tempo Tempo) *Sequence {
	s := &Sequence{
		sample: sample,
		on:     pattern,
		tempo:  tempo,
	}
	var err error
	s.streamer, err = s.getStreamer()
	if err != nil {
		log.Fatal(err)
	}

	return s
}

func (seq *Sequence) getStreamer() (_ beep.Streamer, err error) {
	hold := seq.sample.GetFormat().SampleRate.N(
		time.Duration(
			seq.tempo.Sixteenth(),
		),
	)

	fmt.Println(hold, seq.sample.GetFormat().SampleRate.D(hold))
	newBuf := beep.NewBuffer(seq.sample.GetFormat())
	for _, play := range seq.on {
		var s beep.Streamer
		if play {
			s = seq.sample.GetStreamer(hold)
		} else {
			s = beep.Silence(hold)
		}

		newBuf.Append(s)
	}

	concat := beep.Loop(-1, newBuf.Streamer(0, newBuf.Len()))

	seq.ctrl = &beep.Ctrl{Streamer: concat, Paused: false}
	seq.volume = &effects.Volume{Streamer: seq.ctrl, Base: 2, Volume: 1, Silent: false}
	return seq.volume, err
}

func (seq *Sequence) Stream(buf [][2]float64) (n int, ok bool) {
	n, ok = seq.streamer.Stream(buf[:])
	if !ok {
		seq.err = seq.streamer.Err()
	}

	return n, ok
}

func (seq *Sequence) Err() error {
	return seq.err
}

type Unit struct{}

func main() {
	kick, err := NewSample("./kick.wav")
	if err != nil {
		log.Fatal(err)
	}

	clap, err := NewSample("./clap.wav")
	if err != nil {
		log.Fatal(err)
	}

	hat, err := NewSample("./hat.wav")
	if err != nil {
		log.Fatal(err)
	}
	ft := kick.GetFormat()

	speaker.Init(ft.SampleRate, ft.SampleRate.N(time.Second/10))

	banger := beep.Mix(
		NewSequence(kick, [16]bool{0: true, 4: true, 8: true, 12: true}, Tempo(120)),
		NewSequence(clap, [16]bool{2: true, 6: true, 10: true, 14: true}, Tempo(120)),
		NewSequence(hat, [16]bool{0: true, 2: true, 3: true, 4: true, 6: true, 7: true, 8: true, 10: true, 11: true, 12: true, 14: true}, Tempo(120)),
	)

	fmt.Printf("%++v\n", banger)

	speaker.Play(banger)
	time.Sleep(time.Second * 4 * 4)
}
