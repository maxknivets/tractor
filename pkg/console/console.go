package console

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"sync"

	ct "github.com/daviddengcn/go-colortext"
)

type Console struct {
	Output  io.Writer
	Padding int

	sync.Mutex
}

var colors = []ct.Color{
	ct.Cyan,
	ct.Yellow,
	ct.Green,
	ct.Magenta,
	ct.Red,
	ct.Blue,
}

func (of *Console) LineReader(wg *sync.WaitGroup, name string, index int, r io.Reader, isError bool) {
	defer wg.Done()

	var color ct.Color
	if index == -1 {
		color = ct.White
	} else {
		color = colors[index%len(colors)]
	}

	reader := bufio.NewReader(r)

	var buffer bytes.Buffer

	for {
		buf := make([]byte, 1024)

		if n, err := reader.Read(buf); err != nil {
			return
		} else {
			buf = buf[:n]
		}

		for {
			i := bytes.IndexByte(buf, '\n')
			if i < 0 {
				break
			}
			buffer.Write(buf[0:i])
			of.WriteLine(name, buffer.String(), color, ct.None, isError)
			buffer.Reset()
			buf = buf[i+1:]
		}

		buffer.Write(buf)
	}
}

// Write out a single coloured line
func (of *Console) WriteLine(left, right string, leftC, rightC ct.Color, isError bool) {
	of.Lock()
	defer of.Unlock()

	ct.ChangeColor(leftC, true, ct.None, false)
	formatter := fmt.Sprintf("%%-%ds | ", of.Padding)
	fmt.Fprintf(of.Output, formatter, left)

	if isError {
		ct.ChangeColor(ct.Red, true, ct.None, true)
	} else {
		ct.ResetColor()
	}
	fmt.Fprintln(of.Output, right)
	if isError {
		ct.ResetColor()
	}

}
