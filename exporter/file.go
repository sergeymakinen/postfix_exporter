package exporter

import (
	"bufio"
	"io"
	"os"

	"github.com/nxadm/tail"
)

// File collects Postfix logs from a file.
type File struct {
	Path string
	Test bool

	tail *tail.Tail
}

func (f *File) Collect(ch chan<- result) error {
	if f.Test {
		return f.read(ch)
	}
	return f.start(ch)
}

func (f *File) start(ch chan<- result) error {
	t, err := tail.TailFile(f.Path, tail.Config{
		Location:  &tail.SeekInfo{Whence: io.SeekEnd},
		ReOpen:    true,
		MustExist: true,
		Follow:    true,
		Logger:    tail.DiscardingLogger,
	})
	if err != nil {
		return err
	}
	f.tail = t
	go func() {
		for s := range f.tail.Lines {
			var res result
			res.rec, res.err = parseRecord(s.Text)
			ch <- res
		}
	}()
	return nil
}

func (f *File) read(ch chan<- result) error {
	ff, err := os.Open(f.Path)
	if err != nil {
		return err
	}
	defer ff.Close()
	scanner := bufio.NewScanner(ff)
	for scanner.Scan() {
		var res result
		res.rec, res.err = parseRecord(scanner.Text())
		ch <- res
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func (f *File) Close() error {
	if f.tail == nil {
		return nil
	}
	defer f.tail.Cleanup()
	return f.tail.Stop()
}
