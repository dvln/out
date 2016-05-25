// This is adapted from a discussion on stack overflow:
//   http://stackoverflow.com/questions/28796021/how-can-i-log-in-golang-to-a-file-with-log-rotation
// Frankly I would recommend using Nate Finch's lumberjack in most cases but
// if looking for  a "builtin" rotator feel free to modify/use/adjust this.
// Note that currently the user must decide when to call Rotate() below.

package out

import (
	"os"
	"sync"
	"time"
)

type RotateWriter struct {
	lock     sync.Mutex
	filename string // should be set to the actual filename
	fp       *os.File
}

// NewRotateWr makes a new RotateWriter.  I would tend to recommend using lumberjack
// if it meets your needs but if you have specific naming or rotation needs outside
// of it's scope feel free to adjust/improve/use this (borrowed from answer on net)
func NewRotateWr(filename string) *RotateWriter {
	w := &RotateWriter{filename: filename}
	err := w.Rotate()
	if err != nil {
		return nil
	}
	return w
}

// Write satisfies the io.Writer interface.
func (w *RotateWriter) Write(output []byte) (int, error) {
	w.lock.Lock()
	defer w.lock.Unlock()
	return w.fp.Write(output)
}

// Perform the actual act of rotating and reopening file.
func (w *RotateWriter) Rotate() error {
	w.lock.Lock()
	defer w.lock.Unlock()

	var err error
	// Close existing file if open
	if w.fp != nil {
		err = w.fp.Close()
		w.fp = nil
		if err != nil {
			return err
		}
	}
	// Rename dest file if it already exists
	_, err = os.Stat(w.filename)
	if err == nil {
		err = os.Rename(w.filename, w.filename+"."+time.Now().Format(time.RFC3339))
		if err != nil {
			return err
		}
	}

	// Create a file.
	w.fp, err = os.Create(w.filename)
	return err
}
