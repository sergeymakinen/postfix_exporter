package exporter

import (
	"io"

	"github.com/nxadm/tail"
)

type fileCollector struct {
	tail *tail.Tail
}

func collectFromFile(path string, f func(r record, err error)) (io.Closer, error) {
	t, err := tail.TailFile(path, tail.Config{
		Location:  &tail.SeekInfo{Whence: io.SeekEnd},
		ReOpen:    true,
		MustExist: true,
		Follow:    true,
		Logger:    tail.DiscardingLogger,
	})
	if err != nil {
		return nil, err
	}
	go func() {
		for s := range t.Lines {
			f(parseRecord(s.Text))
		}
	}()
	return &fileCollector{tail: t}, nil
}

func (c *fileCollector) Close() error {
	defer c.tail.Cleanup()
	return c.tail.Stop()
}
