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

package out

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync/atomic"
)

var (
	// defaultErrCode ties into assigning an error code to all errors so if
	// you aren't using codes (or haven't set them in some err scenarios, which
	// can be normal unless you're applying codes to and wrapping all errors
	// which is unlikely).  Anyhow, the pkg will use this default error code
	// for any error that has no code (mostly internal, if this is an errors
	// code it will not be shown typically)
	defaultErrCode int32 = 100
)

// DetailedError (interface) exposes additional information about a BaseError.
// One does not need to use such errors to use the 'out' package (at all) but
// by doing so one can perhaps have more detailed error information available
// for clients (or admin troubleshooting) if preferred.  This allows errors
// to be stacked, stack traces to be stashed, etc.  Basically this interface
// exposes details about 'out' package errors (& implements Go error interface).
type DetailedError interface {
	// Message returns the error message without the stack trace.
	Message() string

	// Stack returns the stack trace without the error message.
	Stack() string

	// This returns the stack trace's context
	Context() string

	// This returns the stack trace without the error message.
	Code() int

	// This returns the wrapped error.  This returns nil if this does not wrap
	// another error.
	Inner() error

	// LvlOut gets the current output level, typically defaults to ERROR but can
	// be any of these levels: ISSUE, ERROR, FATAL
	LvlOut() *LvlOutput

	// SetLvlOut will set the errors leveled output structure to what is given
	SetLvlOut(lvlOut *LvlOutput)

	// Implements the Go built-in error interface.
	Error() string
}

// BaseError can be used for fancier erroring (not required).  This pkg will
// take advantage of such errors if used so that stack traces dumped are as
// close to the originating error as possible and all error messages as errors
// are "passed down" are made visible in the error message (most recent to
// original error message which may be a basic Go error)... this is a nested
// error structure based on work from Dropbox (thanks folks!).  On top of this
// the BaseError, which implements the DetailedError interface, is also hooked
// into the 'out' packages level so if the error is dumped directly it will
// honor 'out' package settings for that output level (defaults to ERROR but
// if you pass an error into Issue() or Fatal() then it will use the leveled
// output appropriate for the log level in use *unless* you use something
// less severe than ISSUE in which case it'll stick with the ERROR level.
type BaseError struct {
	msg     string
	err     error
	code    int
	stack   string
	context string
	inner   error
	lvlOut  *LvlOutput
}

// DefaultErrCode gets the current default error code if you're using
// the DetailedError mechanism.  If not then don't worry about it.  Please
// pass in an int32 (the starting default is 100).
func DefaultErrCode() int32 {
	return defaultErrCode
}

// SetDefaultErrCode can change the default error code so if you want your
// apps default error code to be 1000 or -1 or whatever you can feel free
// to tweak the default error code.  Do not use 0, it will be ignored as
// 0 has special meaning and can't be used.  Aside: currently the default
// error code isn't shown (either 0 or the default aren't shown, but if
// you want that let me know and we'll add in a way to flip that on).
func SetDefaultErrCode(code int32) {
	if code == 0 {
		return
	}
	atomic.StoreInt32(&defaultErrCode, code)
}

// Message returns the error string without stack trace information, note
// that this will recurse across all nested errors whereas the use of something
// like "detErr.Message()" would only return the message *in* that one error
// even if it was part of a set of nested/inner errors.
func Message(err interface{}) string {
	switch e := err.(type) {
	case DetailedError:
		detErr := DetailedError(e)
		ret := []string{}
		for detErr != nil {
			ret = append(ret, detErr.Message())
			i := detErr.Inner()
			if i == nil {
				break
			}
			var ok bool
			detErr, ok = i.(DetailedError)
			if !ok {
				ret = append(ret, i.Error())
				break
			}
		}
		return strings.Join(ret, "\n")
	case runtime.Error:
		return runtime.Error(e).Error()
	case error:
		return e.Error()
	default:
		return "Passed a non-error to Message"
	}
}

// Code returns the errors code (if no code, ie: code=0, then the "default"
// error code (100, as set in defaultErrCode) will be returned.  This routine
// will recurse across all nested/inner errors, the basics:
// a) the "most outer" code that is not 0 or defaultErrCode (if 3 nested errors
// and the middle is set to 209 and the rest aren't set, ie: 0 or defaultErrCode
// then 209 will be the err code returned)
// b) if nothing is set then return the defaultErrCode (typically 100)
// This is different than "detErr.Code()" as that will get whatever code
// is set for that specific error only (will not recurse inner errors/etc)
func Code(err interface{}) int {
	switch e := err.(type) {
	case DetailedError:
		detErr := DetailedError(e)
		code := 0
		for detErr != nil {
			code = detErr.Code()
			if code != 0 && code != int(defaultErrCode) {
				break
			}
			i := detErr.Inner()
			if i == nil {
				break
			}
			var ok bool
			detErr, ok = i.(DetailedError)
			if !ok {
				break
			}
		}
		if code == 0 {
			code = int(defaultErrCode)
		}
		return code
	default:
		return int(defaultErrCode)
	}
}

// Error returns a string with all available error information, including inner
// errors that are wrapped by this errors and a stack trace.
// Note: If you need more flexibility (don't want stack trace, don't want
// all errors messages from the error "stack" then see DefaultError() API)
func (e *BaseError) Error() string {
	stackTrace := false
	shallow := false
	prefix := false
	return DefaultError(e, stackTrace, shallow, prefix)
}

// Message returns the error message without the stack trace.  Note that
// this will not recurse inner/nested errors at all, see "Message(someErr)"
// for that functionality (vs. this being called via "detErr.Message()")
func (e *BaseError) Message() string {
	return e.msg
}

// Stack returns the stack trace without the error message.
func (e *BaseError) Stack() string {
	return e.stack
}

// Code returns the code, if any, available in the given error... note that
// this will not recurse inner/nested errors at all, see "Code(someErr)" for
// that functionality (vs. this being called via "detErr.Code()")
func (e *BaseError) Code() int {
	if e.code == 0 {
		e.code = int(defaultErrCode)
	}
	return e.code
}

// Context returns the stack trace's context.
func (e *BaseError) Context() string {
	return e.context
}

// Inner returns the wrapped error, if there is one.
func (e *BaseError) Inner() error {
	return e.inner
}

// LvlOut returns the currently configured output level struct
func (e *BaseError) LvlOut() *LvlOutput {
	if e.lvlOut == nil {
		e.lvlOut = ERROR
	}
	return e.lvlOut
}

// SetLvlOut returns the currently configured output level struct, note
// that error level out must be ISSUE, ERROR or FATAL otherwise this will
// revert to using ERROR as the output level for this error
func (e *BaseError) SetLvlOut(lvlOut *LvlOutput) {
	if lvlOut.level < LevelIssue {
		e.lvlOut = ERROR
	} else {
		e.lvlOut = lvlOut
	}
}

// NewErr returns a new BaseError initialized with the given message and
// the current stack trace.
func NewErr(msg string, code ...int) DetailedError {
	stack, context := stackTrace(2)
	errNum := 0
	if code != nil {
		errNum = code[0]
	}
	return &BaseError{
		msg:     msg,
		code:    errNum,
		stack:   stack,
		context: context,
		lvlOut:  ERROR,
	}
}

// NewErrf is the same as Err, but with fmt.Printf-style params and error
// code # required
func NewErrf(code int, format string, args ...interface{}) DetailedError {
	stack, context := stackTrace(2)
	return &BaseError{
		msg:     fmt.Sprintf(format, args...),
		code:    code,
		stack:   stack,
		context: context,
		lvlOut:  ERROR,
	}
}

// WrapErr wraps another error in a new BaseError.
func WrapErr(err error, msg string, code ...int) DetailedError {
	stack, context := stackTrace(2)
	errNum := 0
	if code != nil {
		errNum = code[0]
	}
	return &BaseError{
		msg:     msg,
		code:    errNum,
		stack:   stack,
		context: context,
		lvlOut:  ERROR,
		inner:   err,
	}
}

// WrapErrf is the same as WrapErr, but with fmt.Printf-style parameters and
// a required error code #
func WrapErrf(err error, code int, format string, args ...interface{}) DetailedError {
	stack, context := stackTrace(2)
	return &BaseError{
		msg:     fmt.Sprintf(format, args...),
		code:    code,
		stack:   stack,
		context: context,
		lvlOut:  ERROR,
		inner:   err,
	}
}

// DefaultError is a default implementation of the Error method of the detailed
// error interface, see "(DetailedError) Error()" in this pkg.  Unlike the
// detailed error "Error()" method this routine has a set of parameters that
// allow one to customize the error msg returned (and it is publicly available).
// The parameters:
// - e: the DetailedError you want an error string for
// - withStackTrace: boolean indicating if you want a stack trace w/the error,
// note that this is the most inner stack trace (offering the most detail)
// - shallow: boolean indicating if you want just the latest error or all errors
// - outLvlPfx: boolean indicating if you want the standard 'out' package error
// outLvlPfx defaults to "Error: " if no code and "Error #<code>: " if code
// is available in the detailed error (non 0 and non-fallback).  Note that if
// you've changed your prefix to "" or something with no ':" in it then the
// error code will not be inserted.
func DefaultError(e DetailedError, withStackTrace, shallow, outLvlPfx bool) string {
	var errLines []string
	var origStack string

	fillErrorInfo(e, shallow, &errLines, &origStack)
	if withStackTrace {
		errLines = append(errLines, "")
		errLines = append(errLines, "Stack Trace: "+origStack)
	}
	result := strings.Join(errLines, "\n")
	if outLvlPfx {
		outLvl := e.LvlOut()
		errCode := Code(e)
		result = InsertPrefix(result, outLvl.prefix, AlwaysInsert, errCode)
	}
	return result
}

// fillErrorInfo fills errLines with all error messages, and origStack with the
// inner-most stack.
func fillErrorInfo(err error, shallow bool, errLines *[]string, origStack *string) {
	if err == nil {
		return
	}

	derr, ok := err.(DetailedError)
	if ok {
		if !shallow || (shallow && len(*errLines) == 0) {
			*errLines = append(*errLines, derr.Message())
		}
		*origStack = derr.Stack()
		fillErrorInfo(derr.Inner(), shallow, errLines, origStack)
	} else {
		if !shallow || (shallow && len(*errLines) == 0) {
			*errLines = append(*errLines, err.Error())
		}
	}
	//TESTING: verify the shallow functionality, add tests
}

// unwrapError returns a wrapped error or nil if there is none.
func unwrapError(ierr error) (nerr error) {
	// Internal errors have a well defined bit of context.
	if detErr, ok := ierr.(DetailedError); ok {
		return detErr.Inner()
	}

	// At this point, if anything goes wrong, just return nil.
	defer func() {
		if x := recover(); x != nil {
			nerr = nil
		}
	}()

	// Go system errors have a convention but paradoxically no
	// interface.  All of these panic on error.
	errV := reflect.ValueOf(ierr).Elem()
	errV = errV.FieldByName("Err")
	return errV.Interface().(error)
}

// RootError keeps peeling away layers or context until a primitive error is
// revealed.
func RootError(ierr error) (nerr error) {
	nerr = ierr
	for i := 0; i < 500; i++ {
		terr := unwrapError(nerr)
		if terr == nil {
			return nerr
		}
		nerr = terr
	}
	return fmt.Errorf("too many iterations: %T", nerr)
}

// MatchingErrCodes keeps peeling away layers of errors to see if any of the
// given error codes (each which should be set to true in the validCodes map)
// are in use in any of the layers of errors... only try 40 deep for now.
func MatchingErrCodes(err error, validCodes map[int]bool) bool {
	errCodeFound := false
	for i := 0; i < 500; i++ {
		nextErr := unwrapError(err)
		if nextErr == nil {
			break
		}
		if detErr, ok := err.(DetailedError); ok {
			currCode := detErr.Code()
			if validCodes[currCode] {
				errCodeFound = true
				break
			}
		}
		err = nextErr
	}
	return errCodeFound
}

// IsError performs a deep check, unwrapping errors as much as possible and
// comparing the string version of the error (as well as having the ability
// to check for valid/set error codes, if they are in use).  The idea is
// that core Go libs and other pkg's often provide error constants so one can
// detect if a given type of error is coming back from a library/pkg.  That
// comparison only works if one has the original "core" library Go error (the
// "root error" in the case of wrapped/nested errors).  As to error codes, with
// a DetailedError one can use error codes... if so one can either pass in
// a error constant or one or more error codes (or both) and any nested err
// that uses a matching code (assuming non-0 and not set to the defaultErrCode
// both of which are "reserved" codes typically meaning "not set or not in use")
// will result in True, ie: it is a matching error, being returned.
func IsError(err, errConst error, codes ...int) bool {
	if errConst == nil && codes == nil {
		return false
	}
	if err == errConst {
		return true
	}
	validCodes := make(map[int]bool)
	if codes != nil {
		for _, val := range codes {
			if val != 0 && val != int(defaultErrCode) {
				validCodes[val] = true
			}
		}
		if MatchingErrCodes(err, validCodes) {
			return true
		}
	}

	if errConst == nil {
		return false
	}
	// Must rely on string equivalence, otherwise a value is not equal
	// to its pointer value.
	rootErrStr := ""
	rootErr := RootError(err)
	if rootErr != nil {
		rootErrStr = rootErr.Error()
	}
	errConstStr := ""
	if errConst != nil {
		errConstStr = errConst.Error()
	}
	return rootErrStr == errConstStr
}
