package build

import (
	"bufio"
	"io"
	"sync"
)

func streamOutput(reader io.Reader, stream string, emit func(string, string), done *sync.WaitGroup) {
	defer done.Done()
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 32*1024), 1024*1024)
	for scanner.Scan() {
		emit(stream, scanner.Text())
	}
}
