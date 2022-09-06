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
