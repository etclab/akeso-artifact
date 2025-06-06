package drat

import (
	"bufio"
	"drat/internal/mu"
)

func ReadChannelFrom(scanner bufio.Scanner, ch chan []byte) {
	for scanner.Scan() {
		s := scanner.Text()
		ch <- []byte(s)
	}

	if err := scanner.Err(); err != nil {
		mu.Die("error: read failed: %v", err)
	}

	defer close(ch)
}
