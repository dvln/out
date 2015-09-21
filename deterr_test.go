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

// Package test for: out/deterr.go
//   This file focuses on testing the detailed error part of the 'out'
//   package.

package out

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"syscall"
	"testing"

	"github.com/dvln/testify/assert"
)

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
	assert.Contains(t, screenBuf.String(), "/out.trySomething")

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
	msg     string
	code    int
	extra   int
	stack   string
	context string
	lvlOut  *LvlOutput
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
func (e databaseError) Message() string {
	return fmt.Sprintf(e.msg, e.code, e.extra)
}
func (e databaseError) Stack() string { return e.stack }
func (e databaseError) Code() int {
	if e.code == 0 {
		e.code = 100
	}
	return e.code
}
func (e databaseError) Extra() int                  { return e.extra }
func (e databaseError) Context() string             { return e.context }
func (e databaseError) Inner() error                { return nil }
func (e databaseError) LvlOut() *LvlOutput          { return e.lvlOut }
func (e databaseError) SetLvlOut(lvlOut *LvlOutput) { e.lvlOut = lvlOut }

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

	if dbError.Extra() != -1 {
		t.Errorf("the dbMsg.extra field in the database error wasn't set to -1")
	}

	if dbError.Code() != 1205 {
		t.Errorf("the dbMsg.code field in the database error wasn't set to 1205")
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
