package util

import (
	"log"
	"os"

	"github.com/faiface/beep"
	"github.com/faiface/beep/wav"
)

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

// Logger is a context-aware logger
type Logger struct {
	prefixes []any
}

// Ctx returns a copy of the logger with the given prefix added after all pre-existing prefixes
func (l Logger) Ctx(prefix string) Logger {
	return Logger{append(l.prefixes, prefix+":")}
}

// Log shares its interface with log.Println
func (l Logger) Log(msgs ...any) {
	log.Println(append(l.prefixes, msgs...)...)
}
