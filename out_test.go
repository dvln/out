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
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"syscall"
	"testing"

	"github.com/dvln/testify/assert"
)

// Example of more standard Go test infra usage (if removing assert)
//	if tracer == nil {
//		t.Error("Return from New should not be nil")
//	}
//	tracer.Trace("Hello trace package.")
//	if buf.String() != "Hello trace package.\n" {
//		t.Errorf("Trace should not write '%s'.", buf.String())
//	}

func ResetOutPkg() {
	// TESTING: this currently doesn't reset or store everything, should
	//          likely do a deep copy of TRACE, DEBUG, etc and restore those
	//          as well as stash all pkg globals and reset them, etc... however,
	//          takes care of the most common ones needed for now
	ResetNewline(true, ForBoth)
	SetThreshold(defaultScreenThreshold, ForScreen)
	SetThreshold(defaultLogThreshold, ForLogfile)
	SetStackTraceConfig(StackTraceExitToLogfile)
	ClearFormatter(LevelAll)
}

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

	currWriter := Writer(LevelFatal, ForScreen)
	if currWriter != os.Stderr {
		t.Error("Current screen writer for fatal error should be os.Stderr, it was not though")
	}
	SetWriter(LevelAll, screenBuf, ForScreen)
	SetWriter(LevelAll, logBuf, ForLogfile)

	SetThreshold(LevelInfo, ForScreen)
	SetThreshold(LevelNote, ForLogfile)

	Trace("trace info")
	Debug("debugging info")
	Verbose("verbose info")
	Info("information")

	// Here we'll test the io.Writer support.  Since our logfile output level is
	// set to Note we can check a few things by dumping the note via io.Writer:
	// a) that logBuf contains out_test.go and TestOutput for file/function
	//    meaning that out caller(<depth>) is good for the io.Writer interface
	//    (regular method calls are also tested in other test routines, right
	//     now the same CallDepth of 5 seems good for both, nice)
	// b) that the GetWriter() method is working to get a usable io.Writer
	//    compatible structure
	// c) that the other write mechanisms work in conjunction with the io.Writer
	//    mechanism such that newline insertion and prefixing only happens once
	//    across the various log levels (currently that's what we want)

	n, err := fmt.Fprintf(NOTE, "%s", "key writer note ")
	if err != nil {
		t.Errorf("Error return from fmt.Fprintf() to NOTE should be nil, it isn't, io.Writer issues? (%d)", n)
	}

	// Now do a quick test of the method as well, and in doing so also
	// test below that the prefix wasn't added between these two entries
	noteWriter := GetWriter(LevelNote)
	n, err = fmt.Fprintf(noteWriter, "%s", "key writer note ")
	if err != nil {
		t.Errorf("Error return from fmt.Fprintf() to noteWriter should be nil, it isn't, io.Writer issues? (%d)", n)
	}

	os.Setenv("PKG_OUT_STACK_TRACE_CONFIG", "both,never")
	Issue("user issue")
	Error("critical error")
	os.Setenv("PKG_OUT_NO_EXIT", "1")
	Fatal("fatal error")
	os.Setenv("PKG_OUT_STACK_TRACE_CONFIG", "")
	// Now reset the most common things for the 'out' pkg so the next test
	// func will operate sanely as if we're coming in fresh
	ResetOutPkg()

	assert.NotContains(t, screenBuf.String(), "trace info")
	assert.NotContains(t, screenBuf.String(), "debugging info")
	assert.NotContains(t, screenBuf.String(), "verbose info")
	assert.Contains(t, screenBuf.String(), "information")
	assert.Contains(t, screenBuf.String(), "key writer note key writer note ")
	assert.Contains(t, screenBuf.String(), "user issue")
	assert.Contains(t, screenBuf.String(), "critical errorfatal error")

	assert.NotContains(t, logBuf.String(), "trace info")
	assert.NotContains(t, logBuf.String(), "debugging info")
	assert.NotContains(t, logBuf.String(), "verbose info")
	assert.NotContains(t, logBuf.String(), "information")
	assert.Contains(t, logBuf.String(), "key writer note key writer note ")
	assert.Contains(t, logBuf.String(), "user issue")
	assert.Contains(t, logBuf.String(), "critical error")
	assert.Contains(t, logBuf.String(), "fatal error")
	assert.Contains(t, logBuf.String(), "out_test.go:")
	assert.Contains(t, logBuf.String(), "TestOutput")
	lines := strings.Split(logBuf.String(), "\n")
	if len(lines) != 2 || (len(lines) == 2 && lines[1] != "") {
		t.Errorf("Number of lines dumped (%d) was greater than the one CR terminated line expected\n  string: \"%s\"", len(lines), logBuf.String())
	}
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
	newFlags := Flags(LevelInfo, ForLogfile)
	if newFlags&Lmicroseconds == 0 || newFlags&LstdFlags == 0 ||
		newFlags&Lshortfile == 0 || newFlags&Lshortfunc == 0 {
		t.Error("Package out flags apparently did not get set as expected")
	}

	Traceln("trace info")
	Debugln("debugging info")
	Verboseln("verbose info")
	Println("information")
	Noteln("key note")
	os.Setenv("PKG_OUT_STACK_TRACE_CONFIG", "logfile,never")
	Issueln("user issue")
	Errorln("critical error")
	os.Setenv("PKG_OUT_NO_EXIT", "1")
	Fatalln("fatal error")
	os.Setenv("PKG_OUT_STACK_TRACE_CONFIG", "")

	// Now reset the most common things for the 'out' pkg so the next test
	// func will operate sanely as if we're coming in fresh
	ResetOutPkg()

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
	SetThreshold(LevelTrace, ForBoth)
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
	os.Setenv("PKG_OUT_STACK_TRACE_CONFIG", "both,never")
	Issuef("%s\n", "user issue")
	Errorf("%s\n", "critical error")
	os.Setenv("PKG_OUT_NO_EXIT", "1")
	os.Setenv("PKG_OUT_STACK_TRACE_CONFIG", "both,nonzeroerrorexit")
	fatalPfx := Prefix(LevelFatal)
	if fatalPfx != "Fatal: " {
		t.Errorf("The fatal prefix appears to be set incorrectly, currently: \"%s\", expected: \"Fatal: \"", fatalPfx)
	}
	SetPrefix(LevelFatal, "FREAKOUT: ")
	Fatalf("%s\n", "fatal error")
	SetPrefix(LevelFatal, "Fatal: ")
	os.Setenv("PKG_OUT_STACK_TRACE_CONFIG", "")

	// Now reset the most common things for the 'out' pkg so the next test
	// func will operate sanely as if we're coming in fresh
	ResetOutPkg()

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
	assert.Contains(t, screenBuf.String(), "FREAKOUT: fatal error\n")
	assert.Contains(t, screenBuf.String(), "Stack Trace:")

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
	assert.Contains(t, logBuf.String(), "Stack Trace:")
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

	// Now reset the most common things for the 'out' pkg so the next test
	// func will operate sanely as if we're coming in fresh
	ResetOutPkg()

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
	SetStackTraceConfig(ForBoth | StackTraceErrorExit)
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
	assert.Contains(t, screenBuf.String(), "Stack Trace:")

	logFileBuf, readerr := ioutil.ReadFile(logFileName)
	if readerr != nil {
		t.Errorf("Failed to read temp file: %s (%v)", logFileName, readerr)
	}
	/*fmt.Println("Log File Name:", logFileName)*/
	rmerr := os.Remove(logFileName)
	if rmerr != nil {
		t.Errorf("Failed to remove temp file: %s (%v)", logFileName, rmerr)
	}

	// Now reset the most common things for the 'out' pkg so the next test
	// func will operate sanely as if we're coming in fresh
	ResetOutPkg()

	assert.Contains(t, string(logFileBuf), "trace info, trace over multiple lines")
	assert.Contains(t, string(logFileBuf), "debugging info")
	assert.Contains(t, string(logFileBuf), "verbose info")
	assert.Contains(t, string(logFileBuf), "information")
	assert.Contains(t, string(logFileBuf), "key note")
	assert.Contains(t, string(logFileBuf), "user issue")
	assert.Contains(t, string(logFileBuf), "critical error")
	assert.Contains(t, string(logFileBuf), "fatal error")
	assert.Contains(t, string(logFileBuf), "Stack Trace:")
}

// The below was adapted from Dropbox's errors.go and errors_test.go
// implementation.  The idea is to get a stack trace and more detailed
// error info as close to the original error occurance as possible (not
// after it's been passed back through various go routines... and have
// that data travel with the error as it's passed back and perhaps
// further wrapped).

func TestStackTrace1(t *testing.T) {
	const testMsg = "test error"
	er := NewErr(testMsg, 1233)
	e := er.(*BaseError)

	if e.Msg != testMsg {
		t.Errorf("error message %s != expected %s", e.Msg, testMsg)
	}

	if strings.Index(e.Stack, "dvln/out/out.go") != -1 {
		t.Error("stack trace generation code should not be in the error stack trace")
	}

	if strings.Index(e.Stack, "TestStackTrace") == -1 {
		t.Error("stack trace must have test code in it")
	}

	// compile-time test to ensure that DetailedError conforms to error interface
	var err error = e
	_ = err

	// Now reset the most common things for the 'out' pkg so the next test
	// func will operate sanely as if we're coming in fresh
	ResetOutPkg()
}

func TestStackTrace2(t *testing.T) {
	// lets capture screen output while mirroring to a log file
	screenBuf := new(bytes.Buffer)
	SetWriter(LevelAll, screenBuf, ForScreen)
	SetThreshold(LevelTrace, ForScreen)
	SetThreshold(LevelDiscard, ForLogfile)

	// lets dump stack traces for *any* issues, errors, fatals...
	SetStackTraceConfig(ForScreen | StackTraceAllIssues)
	Issue("user issue\n")

	// Now reset the most common things for the 'out' pkg so the next test
	// func will operate sanely as if we're coming in fresh
	ResetOutPkg()

	assert.Contains(t, screenBuf.String(), "Issue: user issue\n")
	assert.Contains(t, screenBuf.String(), "Stack Trace:")
	assert.Contains(t, screenBuf.String(), "dvln/lib/out.TestStackTrace2")
}

func TestDefaultErrorWithErrCode(t *testing.T) {
	const testMsg = "test error"
	er := NewErr(testMsg, 1233)
	errStr := DefaultError(er, true, false, true)
	if strings.Index(errStr, "Error #1233: test error") == -1 {
		t.Error("Failed to find valid error code in msg")
	}
	if strings.Index(errStr, "Error #1233: Stack Trace: goroutine") == -1 {
		t.Error("Failed to find valid error code with stack trace header in msg")
	}
	er = NewErr(testMsg, int(defaultErrCode))
	errStr = DefaultError(er, true, false, true)
	if strings.Index(errStr, "Error: test error") == -1 {
		t.Error("Failed to handle default error code error msg correctly")
	}
	if strings.Index(errStr, "Error: Stack Trace: goroutine") == -1 {
		t.Error("Failed to handle default error code stack trace correctly")
	}

	// Now reset the most common things for the 'out' pkg so the next test
	// func will operate sanely as if we're coming in fresh
	ResetOutPkg()
}

func lowLevelErr() error {
	return fmt.Errorf("%s", "this is a low level err")
}

func trySomething() error {
	lowErr := lowLevelErr()
	return WrapErr(lowErr, "middle level error", 2210)
}

func TestDetailError(t *testing.T) {
	// Lets see up a new detailed error, woohoo
	midErr := trySomething()
	const testMsg = "test error"
	topErr := WrapErr(midErr, testMsg, 293)

	// lets capture screen output while mirroring to a log file
	screenBuf := new(bytes.Buffer)
	SetWriter(LevelAll, screenBuf, ForScreen)
	SetThreshold(LevelTrace, ForScreen)
	Discard(ForLogfile)
	SetFlags(LevelAll, 0, ForScreen)
	ResetNewline(true, ForBoth)

	Issue(topErr)

	assert.Contains(t, screenBuf.String(), "Issue #293: test error")
	assert.NotContains(t, screenBuf.String(), "Stack Trace:")
	assert.NotContains(t, screenBuf.String(), "dvln/lib/out.TestDetailError")

	screenBuf = new(bytes.Buffer)
	SetWriter(LevelAll, screenBuf, ForScreen)

	ResetNewline(true, ForBoth)
	SetStackTraceConfig(ForScreen | StackTraceAllIssues)
	Issue(topErr)

	assert.Contains(t, screenBuf.String(), "Issue #293: test error")
	assert.Contains(t, screenBuf.String(), "Issue #293: Stack Trace:")
	assert.Contains(t, screenBuf.String(), "Issue #293: dvln/lib/out.trySomething")

	// Now reset the most common things for the 'out' pkg so the next test
	// func will operate sanely as if we're coming in fresh
	ResetOutPkg()
}

func TestWrappedError(t *testing.T) {
	const (
		innerMsg  = "I am inner error"
		middleMsg = "I am the middle error"
		outerMsg  = "I am the mighty outer error"
	)
	inner := fmt.Errorf(innerMsg)
	middle := WrapErr(inner, middleMsg, 400)
	outer := WrapErr(middle, outerMsg, 200)
	errorStr := outer.Error()

	// Now reset the most common things for the 'out' pkg so the next test
	// func will operate sanely as if we're coming in fresh
	ResetOutPkg()

	if strings.Index(errorStr, innerMsg) == -1 {
		t.Errorf("couldn't find inner error message in:\n%s", errorStr)
	}

	if strings.Index(errorStr, middleMsg) == -1 {
		t.Errorf("couldn't find middle error message in:\n%s", errorStr)
	}

	if strings.Index(errorStr, outerMsg) == -1 {
		t.Errorf("couldn't find outer error message in:\n%s", errorStr)
	}

	if IsError(outer, nil, 300) != false {
		t.Errorf("invalid error code matched when it shouldn't have")
	}

	if IsError(outer, nil, 200) == false {
		t.Errorf("valid error code (200) failed to match when it should have")
	}

	if IsError(outer, nil, 300, 400) == false {
		t.Errorf("valid error code (400) failed to match when it should have")
	}

	if IsError(outer, inner, 300) == false {
		t.Errorf("valid error message failed to match nested msgs correctly:\n%s", inner)
	}
	if IsError(outer, fmt.Errorf("%s", "your mama"), 300) == true {
		t.Errorf("invalid error message and code matched nested msgs, shouldn't have:\n%s", errorStr)
	}
}

// ---------------------------------------
// minimal example + test for custom error
//
type databaseError struct {
	Msg     string
	Code    int
	Extra   int
	Stack   string
	Context string
	LvlOut  *LvlOutput
}

// "constructor" for creating error (needs to store return value of
// stackTrace() to get the stack and any context)
func newDatabaseError(msg string, code int, extra int) databaseError {
	stack, context := stackTrace(2)
	return databaseError{msg, code, extra, stack, context, ERROR}
}

// needed to satisfy "error" interface
func (e databaseError) Error() string {
	withStackTrace := false
	shallow := false
	prefix := false
	return DefaultError(e, withStackTrace, shallow, prefix)
}

// for the DetailedError interface
func (e databaseError) GetMessage() string {
	return fmt.Sprintf(e.Msg, e.Code, e.Extra)
}
func (e databaseError) GetStack() string { return e.Stack }
func (e databaseError) GetCode() int {
	if e.Code == 0 {
		e.Code = 100
	}
	return e.Code
}
func (e databaseError) GetExtra() int               { return e.Extra }
func (e databaseError) GetContext() string          { return e.Context }
func (e databaseError) GetInner() error             { return nil }
func (e databaseError) GetLvlOut() *LvlOutput       { return e.LvlOut }
func (e databaseError) SetLvlOut(lvlOut *LvlOutput) { e.LvlOut = lvlOut }

// ---------------------------------------

func TestCustomError(t *testing.T) {
	dbMsg := "database error %d [%d] (lock wait time exceeded)"
	dbMsgFinal := "database error 1205 [-1] (lock wait time exceeded)"
	outerMsg := "outer msg"

	dbError := newDatabaseError(dbMsg, 1205, -1)
	outerError := WrapErr(dbError, outerMsg)

	screenBuf := new(bytes.Buffer)
	SetWriter(LevelAll, screenBuf, ForScreen)
	SetThreshold(LevelTrace, ForScreen)
	SetThreshold(LevelDiscard, ForLogfile)
	SetStackTraceConfig(ForScreen | StackTraceAllIssues)

	// OK, redirected the 'out' package into a buffer and adjusted output
	// thresholds for our test (to screen), turned on stack tracing (for
	// screen), made sure newline tracing was happy so lets fire up an Error():
	Error(outerError)
	// and grab that error from the buffer and check it out
	errorStr := screenBuf.String()

	// Now reset the most common things for the 'out' pkg so the next test
	// func will operate sanely as if we're coming in fresh
	ResetOutPkg()

	if strings.Index(errorStr, dbMsgFinal) == -1 {
		t.Errorf("couldn't find database error message (%s) in:\n%s", dbMsgFinal, errorStr)
	}

	if strings.Index(errorStr, outerMsg) == -1 {
		t.Errorf("couldn't find outer error message in:\n%s", errorStr)
	}

	if strings.Index(errorStr, "out.TestCustomError") == -1 {
		t.Errorf("couldn't find this function in stack trace:\n%s", errorStr)
	}

	if dbError.GetExtra() != -1 {
		t.Errorf("the dbMsg.Extra field in the database error wasn't set to -1")
	}

	if dbError.GetCode() != 1205 {
		t.Errorf("the dbMsg.Code field in the database error wasn't set to 1205")
	}
}

type killScreenOut struct{}
type replaceMsg struct{}
type detectDying struct{}

// FormatMessage in this context is to test the formatting "feature" of
// the 'out' package.
func (f killScreenOut) FormatMessage(msg string, outLevel Level, code int, stack string, dying bool) (string, int, bool) {
	suppressOutputMask := ForScreen
	suppressNativePrefixing := false
	return msg, suppressOutputMask, suppressNativePrefixing
}

// FormatMessage in this context is to test the formatting "feature" of
// the 'out' package to see if it will suppress the
func (f replaceMsg) FormatMessage(msg string, outLevel Level, code int, stack string, dying bool) (string, int, bool) {
	msg = "Replacement message, joy joy joy"
	suppressOutputMask := ForLogfile
	suppressNativePrefixing := true
	return msg, suppressOutputMask, suppressNativePrefixing
}

// FormatMessage in this context is to test the formatting "feature" of
// the 'out' package.
func (f detectDying) FormatMessage(msg string, outLevel Level, code int, stack string, dying bool) (string, int, bool) {
	if dying {
		msg = fmt.Sprintf("Looks like we are dying [DYING #%d]", code)
	}
	suppressOutputMask := 0
	suppressNativePrefixing := false
	return msg, suppressOutputMask, suppressNativePrefixing
}

func TestFormatter(t *testing.T) {
	// Aside: if you want to see nested error messages one could create errors
	// something like this for each level (ie: extend DetailedError with your
	// own error so all detailed errors append " [#<errcode>]" for example to
	// error lines... here we'll just shove more data into the db message
	dbMsg := "database error %d [%d] (lock wait time exceeded)"
	outerMsg := "outer msg"

	dbError := newDatabaseError(dbMsg, 1205, -1)
	outerError := WrapErr(dbError, outerMsg)

	screenBuf := new(bytes.Buffer)
	logfileBuf := new(bytes.Buffer)
	SetWriter(LevelAll, screenBuf, ForScreen)
	SetWriter(LevelAll, logfileBuf, ForLogfile)
	SetThreshold(LevelTrace, ForScreen)
	SetThreshold(LevelTrace, ForLogfile)
	SetStackTraceConfig(ForBoth | StackTraceAllIssues)
	var screenOutOff killScreenOut
	SetFormatter(LevelAll, screenOutOff)

	// OK, redirected the 'out' package into a buffer and adjusted output
	// thresholds for our test (to screen), turned on stack tracing (for
	// screen)
	Error(outerError)
	// Now grab that error from the buffer and check it out
	screenErrStr := screenBuf.String()
	logfileErrStr := logfileBuf.String()

	// At this point we have set a formatter that should prevent any screen
	// buffer output (see SetFormatter() call above), lets make sure we don't
	// see the error or a stack trace in the screen output
	assert.NotContains(t, screenErrStr, "Error #1205:")
	assert.NotContains(t, screenErrStr, "Stack Trace:")

	// But we have enabled logfile writer (buffer) output with stack
	// tracing on so we should see the log file output of that data:
	assert.Contains(t, logfileErrStr, "Error #1205:")
	assert.Contains(t, logfileErrStr, "Stack Trace:")

	// Now reset the most common things for the 'out' pkg so the next test
	// func will operate sanely as if we're coming in fresh
	ResetOutPkg()

	screenBuf = new(bytes.Buffer)
	SetWriter(LevelAll, screenBuf, ForScreen)
	SetThreshold(LevelTrace, ForScreen)
	SetThreshold(LevelDiscard, ForLogfile)
	var screenReplaceMsg replaceMsg
	SetFormatter(LevelAll, screenReplaceMsg)

	// OK, redirected the 'out' package into a buffer and adjusted output
	// thresholds for our test (to screen)
	Error(outerError)

	// Grab that error from the buffer and check it out
	screenErrStr = screenBuf.String()
	assert.NotContains(t, screenErrStr, "Stack Trace:")
	assert.NotContains(t, screenErrStr, "Error #1205:")
	assert.Contains(t, screenErrStr, "Replacement message, joy joy joy")

	// Now reset the most common things for the 'out' pkg so the next test
	// func will operate sanely as if we're coming in fresh
	ResetOutPkg()

	screenBuf = new(bytes.Buffer)
	SetWriter(LevelAll, screenBuf, ForScreen)
	SetThreshold(LevelTrace, ForScreen)
	Discard(ForLogfile)
	var screenDetectDying detectDying
	SetFormatter(LevelAll, screenDetectDying)

	// OK, redirected the 'out' package into a buffer and adjusted output
	// thresholds for our test (to screen)
	os.Setenv("PKG_OUT_NO_EXIT", "1")
	ErrorExit(-1, outerError)
	os.Setenv("PKG_OUT_NO_EXIT", "0")

	// Grab that error from the buffer and check it out
	screenErrStr = screenBuf.String()
	assert.NotContains(t, screenErrStr, "Stack Trace:")
	assert.Contains(t, screenErrStr, "Looks like we are dying [DYING #1205]")

	// Reset the most common things for the 'out' pkg so the next test
	// func will operate sanely as if we're coming in fresh
	ResetOutPkg()
}

func TestSettingVals(t *testing.T) {
	origVal := ErrorExitVal()
	if origVal != errorExitVal {
		t.Errorf("Default error exit value is not %d but should be", errorExitVal)
	}
	var newVal int32
	newVal = 1
	SetErrorExitVal(newVal)
	newVal = ErrorExitVal()
	if newVal != 1 {
		t.Errorf("Setting new error exit val to 1 appears to have failed, found: %d", newVal)
	}
	SetErrorExitVal(errorExitVal)

	origVal = DefaultErrCode()
	if origVal != defaultErrCode {
		t.Errorf("Default error code is not %d as expected", defaultErrCode)
	}
	newVal = 1000
	SetDefaultErrCode(newVal)
	newVal = DefaultErrCode()
	if newVal != 1000 {
		t.Errorf("Setting new default error code to 1000 appears to have failed, found: %d", newVal)
	}
	SetDefaultErrCode(defaultErrCode)

	origVal = CallDepth()
	if origVal != callDepth {
		t.Errorf("Default call depth is not %d as expected", callDepth)
	}
	newVal = 6
	SetCallDepth(newVal)
	newVal = CallDepth()
	if newVal != 6 {
		t.Errorf("Setting new call depth to 6 appears to have failed, found: %d", newVal)
	}
	SetCallDepth(callDepth)

	origVal = ShortFileNameLength()
	if origVal != shortFileNameLength {
		t.Errorf("Default short file name length is not %d as expected, found: %d", shortFileNameLength, origVal)
	}
	newVal = 30
	SetShortFileNameLength(newVal)
	newVal = ShortFileNameLength()
	if newVal != 30 {
		t.Errorf("Setting short file name length to 30 appears to have failed, found: %d", newVal)
	}
	SetShortFileNameLength(shortFileNameLength)

	origVal = LongFileNameLength()
	if origVal != longFileNameLength {
		t.Errorf("Default long file name length is not %d as expected, found: %d", longFileNameLength, origVal)
	}
	newVal = 69
	SetLongFileNameLength(newVal)
	newVal = LongFileNameLength()
	if newVal != 69 {
		t.Errorf("Setting long file name length to 69 appears to have failed, found: %d", newVal)
	}
	SetLongFileNameLength(longFileNameLength)

	origVal = ShortFuncNameLength()
	if origVal != shortFuncNameLength {
		t.Errorf("Default short Func name length is not %d as expected, found: %d", shortFuncNameLength, origVal)
	}
	newVal = 30
	SetShortFuncNameLength(newVal)
	newVal = ShortFuncNameLength()
	if newVal != 30 {
		t.Errorf("Setting short Func name length to 30 appears to have failed, found: %d", newVal)
	}
	SetShortFuncNameLength(shortFuncNameLength)

	origVal = LongFuncNameLength()
	if origVal != longFuncNameLength {
		t.Errorf("Default long Func name length is not %d as expected, found: %d", longFuncNameLength, origVal)
	}
	newVal = 69
	SetLongFuncNameLength(newVal)
	newVal = LongFuncNameLength()
	if newVal != 69 {
		t.Errorf("Setting long Func name length to 69 appears to have failed, found: %d", newVal)
	}
	SetLongFuncNameLength(longFuncNameLength)
}

func TestLevelConversion(t *testing.T) {
	traceStr := fmt.Sprintf("%s", LevelTrace)
	traceLvl := LevelString2Level(traceStr)
	if traceLvl != LevelTrace {
		t.Errorf("Failed to map trace level to string and back")
	}

	errorStr := fmt.Sprintf("%s", LevelError)
	errorLvl := LevelString2Level(errorStr)
	if errorLvl != LevelError {
		t.Errorf("Failed to map error level to string and back")
	}
}

type customErr struct {
}

func (ce *customErr) Error() string { return "testing error" }

type customNestedErr struct {
	Err interface{}
}

func (cne *customNestedErr) Error() string { return "nested testing error" }

func TestRootError(t *testing.T) {
	err := RootError(nil)
	if err != nil {
		t.Fatalf("expected nil error")
	}
	var ce *customErr
	err = RootError(ce)
	if err != ce {
		t.Fatalf("expected err on invalid nil-ptr custom error %T %v", err, err)
	}
	ce = &customErr{}
	err = RootError(ce)
	if err != ce {
		t.Fatalf("expected err on valid custom error")
	}

	cne := &customNestedErr{}
	err = RootError(cne)
	if err != cne {
		t.Fatalf("expected err on empty custom error: %T %v", err, err)
	}

	cne = &customNestedErr{reflect.ValueOf(ce).Pointer()}
	err = RootError(cne)
	if err != cne {
		t.Fatalf("expected err on invalid nested uniptr: %T %v", err, err)
	}

	cne = &customNestedErr{ce}
	err = RootError(cne)
	if err != ce {
		t.Fatalf("expected ce on valid nested error: %T %v", err, err)
	}

	cne = &customNestedErr{ce}
	err = RootError(syscall.ECONNREFUSED)
	if err != syscall.ECONNREFUSED {
		t.Fatalf("expected ECONNREFUSED on valid nested error: %T %v", err, err)
	}
}

func TestLogfileNameSet(t *testing.T) {
	currFileName := LogFileName()
	tmpFileName := UseTempLogFile("dvln")
	newFileName := LogFileName()
	if newFileName != tmpFileName {
		t.Errorf("Temp log file name setting or retrieving broken, found: \"%s\", but expected: \"%s\"", newFileName, tmpFileName)
	}
	logFileName = currFileName
}
