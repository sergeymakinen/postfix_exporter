package exporter

import (
	"bufio"
	"io"
	"os"
	"sync"

	"github.com/nxadm/tail"
)

// File collects Postfix logs from a file.
type File struct {
	Path string
	Test bool

	tail   *tail.Tail
	closed bool
	done   chan struct{}
	wg     sync.WaitGroup
}

func (f *File) Collect(ch chan<- result) error {
	f.done = make(chan struct{})
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
	f.wg.Add(1)
	go func() {
		defer f.wg.Done()
		for {
			select {
			case s := <-f.tail.Lines:
				var res result
				res.rec, res.err = parseRecord(s.Text)
				select {
				case ch <- res:
				case <-f.done:
					return
				}
			case <-f.done:
				return
			}
		}
	}()
	return nil
}

func (f *File) read(ch chan<- result) error {
	ff, err := os.Open(f.Path)
	if err != nil {
		return err
	}
	f.wg.Add(1)
	go func() {
		defer f.wg.Done()
		defer ff.Close()
		scanner := bufio.NewScanner(ff)
		for scanner.Scan() {
			if f.closed {
				return
			}
			var res result
			res.rec, res.err = parseRecord(scanner.Text())
			select {
			case ch <- res:
			case <-f.done:
				return
			}
		}
	}()
	return nil
}

func (f *File) Wait() {
	f.wg.Wait()
}

func (f *File) Close() error {
	f.closed = true
	close(f.done)
	var err error
	if f.tail != nil {
		defer f.tail.Cleanup()
		err = f.tail.Stop()
	}
	return err
}
