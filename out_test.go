// Copyright © 2015 Erik Brady <brady@dvln.org>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package test: out
//      Simple testing for the 'out' package ... could use more around testing
//      new prefix setting, adjusting of formats, adding in checks to make sure
//      the date/time is in the log output but not the screen output

package out

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Example of more standard Go test infra usage (if removing assert)
//	if tracer == nil {
//		t.Error("Return from New should not be nil")
//	}
//	tracer.Trace("Hello trace package.")
//	if buf.String() != "Hello trace package.\n" {
//		t.Errorf("Trace should not write '%s'.", buf.String())
//	}

func TestLevels(t *testing.T) {
	SetThreshold(LevelIssue, ForScreen)
	assert.Equal(t, Threshold(ForScreen), LevelIssue)
	SetThreshold(LevelError, ForLogfile)
	assert.Equal(t, Threshold(ForLogfile), LevelError)
	assert.NotEqual(t, Threshold(ForScreen), LevelError)
	SetThreshold(LevelNote, ForScreen)
	assert.Equal(t, Threshold(ForScreen), LevelNote)
}

func TestOutput(t *testing.T) {
	// first we'll test the <Level>() functions
	screenBuf := new(bytes.Buffer)
	logBuf := new(bytes.Buffer)

	SetWriter(LevelAll, screenBuf, ForScreen)
	SetWriter(LevelAll, logBuf, ForLogfile)

	SetThreshold(LevelInfo, ForScreen)
	SetThreshold(LevelNote, ForLogfile)

	Trace("trace info")
	Debug("debugging info")
	Verbose("verbose info")
	Info("information")
	Note("key note")
	Issue("user issue")
	Error("critical error")
	os.Setenv("PKG_OUT_NO_EXIT", "1")
	Fatal("fatal error")

	assert.NotContains(t, screenBuf.String(), "trace info")
	assert.NotContains(t, screenBuf.String(), "debugging info")
	assert.NotContains(t, screenBuf.String(), "verbose info")
	assert.Contains(t, screenBuf.String(), "information")
	assert.Contains(t, screenBuf.String(), "key note")
	assert.Contains(t, screenBuf.String(), "user issue")
	assert.Contains(t, screenBuf.String(), "critical error")
	assert.Contains(t, screenBuf.String(), "fatal error")

	assert.NotContains(t, logBuf.String(), "trace info")
	assert.NotContains(t, logBuf.String(), "debugging info")
	assert.NotContains(t, logBuf.String(), "verbose info")
	assert.NotContains(t, logBuf.String(), "information")
	assert.Contains(t, logBuf.String(), "key note")
	assert.Contains(t, logBuf.String(), "user issue")
	assert.Contains(t, logBuf.String(), "critical error")
	assert.Contains(t, logBuf.String(), "fatal error")
}

func TestOutputln(t *testing.T) {
	// now we'll test the <Level>ln() functions
	screenBuf := new(bytes.Buffer)
	logBuf := new(bytes.Buffer)

	SetWriter(LevelAll, screenBuf, ForScreen)
	SetWriter(LevelAll, logBuf, ForLogfile)

	SetThreshold(LevelVerbose, ForScreen)
	SetThreshold(LevelNote, ForLogfile)

	SetFlags(LevelAll, LstdFlags|Lmicroseconds|Lshortfile|Lshortfunc, ForLogfile)

	Traceln("trace info")
	Debugln("debugging info")
	Verboseln("verbose info")
	Println("information")
	Noteln("key note")
	Issueln("user issue")
	Errorln("critical error")
	os.Setenv("PKG_OUT_NO_EXIT", "1")
	Fatalln("fatal error")

	assert.NotContains(t, screenBuf.String(), "trace info\n")
	assert.NotContains(t, screenBuf.String(), "debugging info\n")
	assert.Contains(t, screenBuf.String(), "verbose info\n")
	assert.Contains(t, screenBuf.String(), "information\n")
	assert.Contains(t, screenBuf.String(), "Note: key note\n")
	assert.Contains(t, screenBuf.String(), "Issue: user issue\n")
	assert.Contains(t, screenBuf.String(), "Error: critical error\n")
	assert.Contains(t, screenBuf.String(), "Fatal: fatal error\n")

	assert.NotContains(t, logBuf.String(), "trace info\n")
	assert.NotContains(t, logBuf.String(), "debugging info\n")
	assert.NotContains(t, logBuf.String(), "verbose info\n")
	assert.NotContains(t, logBuf.String(), "information\n")
	assert.NotContains(t, logBuf.String(), "out.TestOutputln")
	assert.Contains(t, logBuf.String(), "key note\n")
	assert.Contains(t, logBuf.String(), "out_test.go:")
	assert.Contains(t, logBuf.String(), "TestOutputln")
	assert.Contains(t, logBuf.String(), "user issue\n")
	assert.Contains(t, logBuf.String(), "critical error\n")
	assert.Contains(t, logBuf.String(), "fatal error\n")
}

func TestOutputf(t *testing.T) {
	// now we'll test the <Level>f() functions
	screenBuf := new(bytes.Buffer)
	logBuf := new(bytes.Buffer)

	SetWriter(LevelAll, screenBuf, ForScreen)
	SetWriter(LevelAll, logBuf, ForLogfile)

	SetThreshold(LevelTrace, ForScreen|ForLogfile)

	SetFlags(LevelAll, LstdFlags|Lmicroseconds|Lshortfile|Llongfunc, ForBoth)

	os.Setenv("PKG_OUT_DEBUG_SCOPE", "boguspkg.")
	Tracef("%s\n", "trace info")
	Debugf("%s\n", "debugging info")
	assert.NotContains(t, screenBuf.String(), "trace info\n")
	assert.NotContains(t, screenBuf.String(), "debugging info\n")

	os.Setenv("PKG_OUT_DEBUG_SCOPE", "out.")
	Tracef("%s\n", "trace info")
	Debugf("%s\n", "debugging info")
	Verbosef("%s\n", "verbose info")
	Printf("%s\n", "information")
	Notef("%s\n", "key note")
	Issuef("%s\n", "user issue")
	Errorf("%s\n", "critical error")
	os.Setenv("PKG_OUT_NO_EXIT", "1")
	os.Setenv("PKG_OUT_NONZERO_EXIT_STACKTRACE", "1")
	SetPrefix(LevelFatal, "PANIC: ")
	Fatalf("%s\n", "fatal error")
	SetPrefix(LevelFatal, "Fatal: ")

	// debug: if you want to look at this in the test output, uncomment:
	/*fmt.Println("Screen output:")
	fmt.Println(screenBuf.String())
	fmt.Println("Logfile output:")
	fmt.Println(logBuf.String()) */

	assert.Contains(t, screenBuf.String(), "trace info\n")
	assert.Contains(t, screenBuf.String(), "debugging info\n")
	assert.Contains(t, screenBuf.String(), "verbose info\n")
	assert.Contains(t, screenBuf.String(), "information\n")
	assert.Contains(t, screenBuf.String(), "Note: key note\n")
	assert.Contains(t, screenBuf.String(), "Issue: user issue\n")
	assert.Contains(t, screenBuf.String(), "Error: critical error\n")
	assert.Contains(t, screenBuf.String(), "PANIC: fatal error\n")
	assert.Contains(t, screenBuf.String(), "stacktrace: stack of")

	assert.Contains(t, logBuf.String(), "trace info\n")
	assert.Contains(t, logBuf.String(), "out_test.go:")
	assert.Contains(t, logBuf.String(), "out.TestOutputf")
	assert.Contains(t, logBuf.String(), "debugging info\n")
	assert.Contains(t, logBuf.String(), "verbose info\n")
	assert.Contains(t, logBuf.String(), "information\n")
	assert.Contains(t, logBuf.String(), "key note\n")
	assert.Contains(t, logBuf.String(), "user issue\n")
	assert.Contains(t, logBuf.String(), "critical error\n")
	assert.Contains(t, logBuf.String(), "fatal error\n")
	assert.Contains(t, logBuf.String(), "stacktrace: stack of")
}

func TestDiscard(t *testing.T) {
	// first we'll test the <Level>() functions
	screenBuf := new(bytes.Buffer)
	logBuf := new(bytes.Buffer)

	SetWriter(LevelAll, screenBuf, ForScreen)
	SetWriter(LevelAll, logBuf, ForLogfile)

	// Turn everything off, see if that flies
	SetThreshold(LevelDiscard, ForScreen|ForLogfile)

	Trace("trace info")
	Debug("debugging info")
	Verbose("verbose info")
	Info("information")
	Note("key note")
	Issue("user issue")
	Error("critical error")
	os.Setenv("PKG_OUT_NO_EXIT", "1")
	Fatal("fatal error")

	assert.NotContains(t, screenBuf.String(), "trace info")
	assert.NotContains(t, screenBuf.String(), "debugging info")
	assert.NotContains(t, screenBuf.String(), "verbose info")
	assert.NotContains(t, screenBuf.String(), "information")
	assert.NotContains(t, screenBuf.String(), "key note")
	assert.NotContains(t, screenBuf.String(), "user issue")
	assert.NotContains(t, screenBuf.String(), "critical error")
	assert.NotContains(t, screenBuf.String(), "fatal error")

	assert.NotContains(t, logBuf.String(), "trace info")
	assert.NotContains(t, logBuf.String(), "debugging info")
	assert.NotContains(t, logBuf.String(), "verbose info")
	assert.NotContains(t, logBuf.String(), "information")
	assert.NotContains(t, logBuf.String(), "key note")
	assert.NotContains(t, logBuf.String(), "user issue")
	assert.NotContains(t, logBuf.String(), "critical error")
	assert.NotContains(t, logBuf.String(), "fatal error")
}

func TestTempFileOutput(t *testing.T) {
	// lets capture screen output while mirroring to a log file
	screenBuf := new(bytes.Buffer)
	SetWriter(LevelAll, screenBuf, ForScreen)
	SetThreshold(LevelTrace, ForScreen)

	// note that this auto-sets up the logfile io.Writer for all logging levels
	logFileName := UseTempLogFile("dvln.")
	// remember logging is LevelDiscard by default, turn all entries on
	SetThreshold(LevelTrace, ForLogfile)

	// test all logging levels, screen will get some, log will get all
	// with log being augmented with date/time and file/line# as well
	Tracef("%s", "trace info, ")
	Tracef("%s\n%s\n", "trace over multiple lines", "trace continued line")
	Debugln("debugging info")
	Verbose("verbose info\n")
	Printf("%s\n", "information")
	Noteln("key note")
	Issue("user issue\n")
	Errorf("%s\n", "critical error")

	// Try the SetStackTrace method instead of the env as above...
	SetStacktraceOnExit(true)
	os.Setenv("PKG_OUT_NO_EXIT", "1")
	Fatalln("fatal error")

	// debug: if you want to look at this in the test output, uncomment:
	//fmt.Println("Screen output:")
	//fmt.Println(screenBuf.String())

	assert.Contains(t, screenBuf.String(), "trace info, trace over multiple lines\n")
	assert.Contains(t, screenBuf.String(), "debugging info\n")
	assert.Contains(t, screenBuf.String(), "verbose info\n")
	assert.Contains(t, screenBuf.String(), "information\n")
	assert.Contains(t, screenBuf.String(), "Note: key note\n")
	assert.Contains(t, screenBuf.String(), "Issue: user issue\n")
	assert.Contains(t, screenBuf.String(), "Error: critical error\n")
	assert.Contains(t, screenBuf.String(), "Fatal: fatal error\n")
	assert.Contains(t, screenBuf.String(), "stacktrace: stack of")

	logFileBuf, readerr := ioutil.ReadFile(logFileName)
	if readerr != nil {
		t.Errorf("Failed to read temp file: %s (%v)", logFileName, readerr)
	}
	/*fmt.Println("Log File Name:", logFileName)*/
	rmerr := os.Remove(logFileName)
	if rmerr != nil {
		t.Errorf("Failed to remove temp file: %s (%v)", logFileName, rmerr)
	}

	assert.Contains(t, string(logFileBuf), "trace info, trace over multiple lines")
	assert.Contains(t, string(logFileBuf), "debugging info")
	assert.Contains(t, string(logFileBuf), "verbose info")
	assert.Contains(t, string(logFileBuf), "information")
	assert.Contains(t, string(logFileBuf), "key note")
	assert.Contains(t, string(logFileBuf), "user issue")
	assert.Contains(t, string(logFileBuf), "critical error")
	assert.Contains(t, string(logFileBuf), "fatal error")
	assert.Contains(t, string(logFileBuf), "stacktrace: stack of")
}
