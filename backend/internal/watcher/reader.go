package watcher

import (
	"bufio"
	"io"
	"os"
)

type LineReader struct {
	path   string
	offset int64
}

func NewLineReader(path string) *LineReader {
	return &LineReader{path: path}
}

func (r *LineReader) ReadNewLines() ([]string, error) {
	file, err := os.Open(r.path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	if info.Size() < r.offset {
		r.offset = 0
	}

	if _, err := file.Seek(r.offset, io.SeekStart); err != nil {
		return nil, err
	}

	var lines []string
	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	offset, err := file.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, err
	}
	r.offset = offset

	return lines, nil
}
