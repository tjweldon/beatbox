package util

import (
	"fmt"
	"log"
	"os"

	"github.com/faiface/beep"
	"github.com/faiface/beep/wav"
)

func Min(a, b int) int { return map[bool]int{true: a, false: b}[ a < b ] }

// IO Plumbing

// BufferSample Takes care of sample buffering, fatal error if the sample cannot
// be loaded
func BufferSample(path string) *beep.Buffer {
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}

	decoded, format, err := wav.Decode(file)
	if err != nil {
		log.Fatal(err)
	}

	buf := beep.NewBuffer(format)
	buf.Append(decoded)
	return buf
}

func Map[T, U any](mapFunc func(T) U, s []T) (out []U) {
	for _, t := range s {
		out = append(out, mapFunc(t))
	}

	return out
}

type LogVolume int

const (
	Silent LogVolume = 1 << iota
	Quieter
	Quiet
	Normal
	Loud
	Louder
	Loudest
)

func (lv LogVolume) String() string {
	switch lv {
	case Silent:
		return "Silent"
	case Quieter:
		return "Quieter"
	case Quiet:
		return "Quiet"
	case Normal:
		return "Normal"
	case Loud:
		return "Loud"
	case Louder:
		return "Louder"
	case Loudest:
		return "Loudest"
	default:
		return fmt.Sprintf("%d", lv)
	}
}

// initialise the log level as silent by default
var filterBelow = func(lv LogVolume) *LogVolume { return &lv }(Silent)

// FilterBelow sets the log level below which messages will not be printed
func (lv LogVolume) FilterBelow() LogVolume {
	*filterBelow = lv
	return lv
}

// Logger is a context-aware logger
type Logger struct {
	prefixes []any
	Volume   LogVolume
}

// Ctx returns a copy of the logger with the given prefix added after all pre-existing prefixes
func (l Logger) Ctx(prefix string) Logger {
	return Logger{append(l.prefixes, prefix+":"), l.Volume}
}

// Vol is like a -v option. A Loud logger will print all messages,
// a Silent one will print none
func (l Logger) Vol(v LogVolume) Logger {
	l.Volume = v
	return l
}

// Log shares its interface with log.Println
func (l Logger) Log(msgs ...any) {
	if l.Volume >= *filterBelow {
		l.prefixes = append([]any{fmt.Sprintf("[%s]", l.Volume)}, l.prefixes...)
		log.Println(append(l.prefixes, msgs...)...)
	}
}
