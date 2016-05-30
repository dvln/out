// Copyright © 2015-2016 Erik Brady <brady@dvln.org>
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

// Package test for: out/formatter.go
//   Testing in this file focuses on the formatter functionality available
//   from the out/formatter.go file in the 'out' pkg.  Basically we set up
//   a few different formatters that exercise some of that capability and
//   insure it functions as expected.  What could go wrong?  ;)  This does
//   use routines from the out_test.go test file to get it's job done.

package out

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/dvln/testify/assert"
)

type killScreenOut struct{}
type replaceMsg struct{}
type detectDying struct{}
type logOnlyFormatMsg struct{}

// FormatMessage in this context is to test the formatting "feature" of
// the 'out' package.  In this case we're suppressing all screen output
func (f killScreenOut) FormatMessage(msg string, outLevel Level, code int, dying bool, mdata FlagMetadata) (string, int, int, bool) {
	applyMask := ForBoth
	suppressOutputMask := ForScreen
	suppressNativePrefixing := false
	return msg, applyMask, suppressOutputMask, suppressNativePrefixing
}

// FormatMessage in this context is to test the formatting "feature" of
// the 'out' package to see if it will suppress the native prefixing
func (f replaceMsg) FormatMessage(msg string, outLevel Level, code int, dying bool, mdata FlagMetadata) (string, int, int, bool) {
	msg = "Replacement message, joy joy joy"
	applyMask := ForBoth
	suppressOutputMask := ForLogfile
	suppressNativePrefixing := true
	return msg, applyMask, suppressOutputMask, suppressNativePrefixing
}

// FormatMessage in this context is to test the formatting "feature" of
// the 'out' package to format only the logging side of the messaging while
// the screen side prints in standard format
func (f logOnlyFormatMsg) FormatMessage(msg string, outLevel Level, code int, dying bool, mdata FlagMetadata) (string, int, int, bool) {
	msg = fmt.Sprintf("Formatted!: \"%s\", metadata:\n\"%+v\"\n", msg, mdata)
	applyMask := ForLogfile
	suppressOutputMask := 0
	suppressNativePrefixing := true
	return msg, applyMask, suppressOutputMask, suppressNativePrefixing
}

// FormatMessage in this context is to test the formatting "feature" of
// the 'out' package.
func (f detectDying) FormatMessage(msg string, outLevel Level, code int, dying bool, mdata FlagMetadata) (string, int, int, bool) {
	if dying {
		msg = fmt.Sprintf("Looks like we are dying [DYING #%d]", code)
	}
	applyMask := ForBoth
	suppressOutputMask := 0
	suppressNativePrefixing := false
	return msg, applyMask, suppressOutputMask, suppressNativePrefixing
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
	logfileBuf = new(bytes.Buffer)
	SetWriter(LevelAll, screenBuf, ForScreen)
	SetWriter(LevelAll, logfileBuf, ForLogfile)
	SetThreshold(LevelTrace, ForScreen)
	SetThreshold(LevelTrace, ForLogfile)
	// This formatter causes logfile data to be formatted, not screen data
	var logFormatMsg logOnlyFormatMsg
	SetFormatter(LevelAll, logFormatMsg)

	// OK, redirected the 'out' package into a buffer and adjusted output
	// thresholds for our test (to screen) and fire up an error:
	Error(outerError)

	// Grab that error from the buffer and check it out
	screenErrStr = screenBuf.String()
	logfileErrStr = logfileBuf.String()

	assert.NotContains(t, screenErrStr, "Formatted!")
	assert.NotContains(t, screenErrStr, "{Time:")
	assert.NotContains(t, screenErrStr, "Func:")
	assert.NotContains(t, screenErrStr, "LineNo:")
	assert.Contains(t, logfileErrStr, "Formatted!")
	assert.Contains(t, logfileErrStr, "{Time:")
	assert.Contains(t, logfileErrStr, "Func:")
	assert.Contains(t, logfileErrStr, "LineNo:")

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
