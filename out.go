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

// Package out is for easy and flexible CLI output and log handling.  It has
// taken ideas from the Go author log package, spf13's jwalterweatherman package,
// Dropbox errors package and other packages out there (many thanks to all the
// talented folks!!!).  Goals of this pkg:
//
// - Leveled output: trace, debug, verbose, print, note, issue, error, fatal
//
// - Out of box for basic screen output (stdout/stderr), any io.Writer supported
//
// - Ability to "mirror" screen output to log file or any other io.Writer
//
// - Ability to dynamically filter debugging output by function or pkg path
//
// - Screen and logfile targets can be independently managed, eg: screen
// gets normal output and errors, log gets full trace/debug output and is
// augmented with timestamps, Go file/line# for all log entry types, etc
//
// - Does not insert carriage returns in output, does cleaner formatting of
// prefixes and meta-data with multiline or non-newline terminated output (vs
// Go's 'log' pkg)
//
// - Non-zero exits can be marked up with a stack trace easily (via env or api)
//
// - Future (partially done, don't use yet): extended errors with smart stack
// tracing, error codes (optional/extensible), error "stacking/wrapping" with
// intelligent "constant error" matching
//
// - Future: Support custom formatters if existing formatting options are not
// desirable, this could be used to dump errors in different formats (eg:
// adjust output if tool running in text or JSON mode for example)
//
// - Future: github.com/dvln/in for prompting/paging, to work w/this package
//
// The 'out' package is designed as a singleton (currently) although one could
// make it more generic... but as I have no need for that currently I've avoided
// that effort.  If done maybe group []*LvlOutput in an "Outputter" struct, add
// methods for all desired functions like 'out.Print()' on (*Outputter) and move
// the logic into that and have the singleton function call these methods.  Then
// perhaps clean up the *Newline stuff (could be done anyhow) so it drives off
// the io.Writers targets (consider os.Stdout and os.Stderr to be the same tgt
// no matter how many writers point at it, and consider any other io.Writer
// like a file or a buffer to be the same if the same "handle"... anyhow, needs
// to be better than what's here now).  What could go wrong?  ;)
//
// Anyhow, for true screen mirroring to logfile type controls it's pretty
// effective as a singleton so have some fun.
//
// Usage:   (Note: each is like 'fmt' syntax for Print, Printf, Println)
//	// For extremely detailed debugging, "<date/time> Trace: " prefix by default
//	out.Trace[f|ln](..)
//
//	// For basic debug output, "<date/time> Debug: " prefix by default to screen
//	out.Debug[f|ln](..)
//
//	// For user wants verbose but still "regular" output, no prefix to screen
//	out.Verbose[f|ln](..)
//
//	// For basic default "normal" output (typically), no prefix to screen
//	out.Print[f|ln](..)    |    out.Info[f|ln](..)   [both do same thing]
//
//	// For key notes for the user to consider (ideally), "Note: " prefix
//	out.Note[f|ln](..)
//
//	// For "expected" usage issues/errors (eg: bad flag value), "Issue: " prefix
//	out.Issue[f|ln](..)
//
//	// For system/setup class error, unexpected errors, "ERROR: " prefix
//	out.Error[f|ln](..)            (default screen out: os.Stderr)
//
//	// For fatal errors, will cause the tool to exit non-zero, "FATAL: " prefix
//	out.Fatal[f|ln](..)            (default screen out: os.Stderr)
//
// Note: logfile format defaults to: <date/time> <shortfile/line#> [Level: ]msg
//
// Aside: for my CLI's options I like "[-d | --debug]" and "[-v | --verbose]"
// to control tool output verbosity, ie: "-dv" (both) is the "output everything"
// mode via the Trace level, just "-d" is the Debug level and all levels below,
// just "-v" sets the Verbose level and all levels below and Info/Print is the
// default devel with none of those options.  Use of [-t | --terse ] maps to
// the Issue and below levels (or whatever you like)... and perhaps "-tv" could
// map to the Note level if you wanted to go that route.  I recommend that the
// viper/cobra packages be used to allow control via CLI, env, config file, etc
// so the user has flexibility in setting their defaults.
package out

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Some of these flags are borrowed from Go's log package and "mostly" behave
// the same but handle multi-line strings and non-newline terminated strings
// differently when adding markup like date/time and file/line# meta-data to
// either screen or log output stream (also better aligns the data overall).
// If date and time with milliseconds is on and long filename w/line# it'll
// look like this:
//   2009/01/23 01:23:23.123123 /a/b/c/d.go:23: [LvlPrefix: ]<mesg>
// If one adds in the pid and level settings it will look like this:
//   [pid] LEVEL 2009/01/23 01:23:23.123123 /a/b/c/d.go:23: [LvlPrefix: ]<mesg>
// And with the flags not on (note that the level prefix depends upon what
// level one is printing output at and it can be adjusted as well):
//   [LvlPrefix: ]<message>
// See SetFlags() below for adjusting settings and Flags() to query settings.
const (
	Ldate         = 1 << iota             // the date: 2009/01/23
	Ltime                                 // the time: 01:23:23
	Lmicroseconds                         // microsecond resolution: 01:23:23.123123.  assumes Ltime.
	Llongfile                             // full file name path and line number: /a/b/c/d.go:23
	Lshortfile                            // just src file name and line #, eg: d.go:23. overrides Llongfile
	Llongfunc                             // full func signature, dvln/cmd.get for get method in dvln/cmd
	Lshortfunc                            // just short func signature, trimmed to just get
	Lpid                                  // add in the pid to the output
	Llevel                                // add in the output level "raw" string (eg: TRACE,DEBUG,..)
	LstdFlags     = Ldate | Ltime         // for those used to Go 'log' flag settings
	LscreenFlags  = Ltime | Lmicroseconds // values for "std" screen and log file flags
	LlogfileFlags = Lpid | Llevel | Ldate | Ltime | Lmicroseconds | Lshortfile | Lshortfunc
)

// Available output and logging levels to this package, by
// default "normal" info output and any notes/issues/errs/fatal/etc
// will be dumped to stdout and, by default, file logging for that output
// is inactive to start with til a log file is set up
const (
	LevelTrace   Level = iota // Very high amount of debug output
	LevelDebug                // Standard debug output level
	LevelVerbose              // Verbose output if user wants it
	LevelInfo                 // Standard output info/print to user
	LevelNote                 // Likely a heads up, "Note: <blah>"
	LevelIssue                // Typically a normal user/usage error
	LevelError                // Recoverable sys or unexpected error
	LevelFatal                // Very bad, we need to exit non-zero
	LevelDiscard              // Indicates no output at all if used
	LevelAll                  // Used for a few API's to indicate set all levels
	// (writer set to ioutil.Discard also works)
	defaultScreenThreshold = LevelInfo    // Default out to regular info level
	defaultLogThreshold    = LevelDiscard // Default file logging starts off
)

// Some API's require an indication of if we're adjusting the "screen" output
// stream or the logfile output stream, or both.  Use these flags as needed
// to indicate which:
const (
	ForScreen  = 1 << iota              // Used to indicate screen output target
	ForLogfile                          // Indicates to control logfile target
	ForBoth    = ForScreen | ForLogfile // indicate both screen/logfile targets
)

// These are primarily for inserting prefixes on printed strings so we can put
// the prefix insert into different modes as needed, see doPrefixing() below.
const (
	AlwaysInsert  = 1 << iota // Prefix every line, regardless of output history
	SmartInsert               // Output context "to now" decides if prefix used
	BlankInsert               // Only spaces inserted (same length as prefix)
	SkipFirstLine             // 1st line in multi-line string has no prefix
)

// Level type is just an int, see related const enum with LevelTrace, ..
type Level int

// LvlOutput structures define io.Writers (eg: one for screen, one for log) to
// flexibly control outputting to a given output "level".  Each writer has a
// set of flags associated indicating what augmentation the output might have
// and there is a single, optional, prefix that will be inserted before any
// message to that level (regardless of screen or logfile).  There are 8 levels
// defined and placed into a array of LvlOutput pointers, []outputters.  Each
// levels output struct screen and log file writers can be individually
// controlled (but would typically all point to stdout/stderr for the screen
// target and the same log file writer or buffer writer for all logfile writers
// for each level... but don't have to).  The log file levels provided are
// currently: trace, debug, verbose, normal, note, issue, error and fatal
// which map to the related singleton functions of the same name, ie:
// Trace[f|ln](), Debug[f|ln](), Verbose[f|ln](), etc.  All prefixes and
// screen handles and such are "bootstrapped" below and can be controlled
// via various methods to change writers, prefixes, overall threshold levels
// and newline tracking, etc.  Aside: below there is a also an io.Writer that
// corresponds to each level, ie: fmt.Fprintf(TRACE, "%s", someStr), as a 2nd
// way to push output through the screen/log writers that are set up.
type LvlOutput struct {
	mu          sync.Mutex // ensures atomic writes; protects these fields:
	level       Level      // below data tells how each logging level works
	prefix      string     // prefix for this logging level (if any)
	buf         []byte     // for accumulating text to write at this level
	screenHndl  io.Writer  // io.Writer for "screen" output
	screenFlags int        // flags: additional metadata on screen output
	logfileHndl io.Writer  // io.Writer for "logfile" output
	logFlags    int        // flags: additional metadata on logfile output
}

// DetailedError (interface) exposes additional information about a BaseError.
// One does not need to use such errors to use the 'out' package (at all) but
// by doing so one can perhaps have more detailed error information available
// for clients (or admin troubleshooting) if preferred.  This allows errors
// to be stacked, stack traces to be stashed, etc.  Basically this interface
// exposes details about 'out' package errors (& implements Go error interface).
type DetailedError interface {
	// This returns the error message without the stack trace.
	GetMessage() string

	// This returns the stack trace without the error message.
	GetStack() string

	// This returns the stack trace's context
	GetContext() string

	// This returns the stack trace without the error message.
	GetCode() int

	// This returns the wrapped error.  This returns nil if this does not wrap
	// another error.
	GetInner() error

	// Implements the Go built-in error interface.
	Error() string
}

// BaseError can be used for fancier errors for your tool, this package will
// take advantage of such errors when formatting errors and such but there
// is no harm in not using these.
type BaseError struct {
	Msg     string
	Err     error
	Code    int
	Stack   string
	Context string
	inner   error
}

var (
	// Set up each output level, ie: level, prefix, screen/log hndl, flags, ...

	// TRACE can be used as an io.Writer for trace level output
	TRACE = &LvlOutput{level: LevelTrace, prefix: "Trace: ", screenHndl: os.Stdout, screenFlags: LscreenFlags, logfileHndl: ioutil.Discard, logFlags: LlogfileFlags}
	// DEBUG can be used as an io.Writer for debug level output
	DEBUG = &LvlOutput{level: LevelDebug, prefix: "Debug: ", screenHndl: os.Stdout, screenFlags: LscreenFlags, logfileHndl: ioutil.Discard, logFlags: LlogfileFlags}
	// VERBOSE can be used as an io.Writer for verbose level output
	VERBOSE = &LvlOutput{level: LevelVerbose, prefix: "", screenHndl: os.Stdout, screenFlags: 0, logfileHndl: ioutil.Discard, logFlags: LlogfileFlags}
	// INFO can be used as an io.Writer for info|print level output
	INFO = &LvlOutput{level: LevelInfo, prefix: "", screenHndl: os.Stdout, screenFlags: 0, logfileHndl: ioutil.Discard, logFlags: LlogfileFlags}
	// NOTE can be used as an io.Writer for note level output
	NOTE = &LvlOutput{level: LevelNote, prefix: "Note: ", screenHndl: os.Stdout, screenFlags: 0, logfileHndl: ioutil.Discard, logFlags: LlogfileFlags}
	// ISSUE can be used as an io.Writer for issue level output
	ISSUE = &LvlOutput{level: LevelIssue, prefix: "Issue: ", screenHndl: os.Stdout, screenFlags: 0, logfileHndl: ioutil.Discard, logFlags: LlogfileFlags}
	// ERR can be used as an io.Writer for error level output
	ERR = &LvlOutput{level: LevelError, prefix: "Error: ", screenHndl: os.Stderr, screenFlags: 0, logfileHndl: ioutil.Discard, logFlags: LlogfileFlags}
	// FATAL can be used as an io.Writer for fatal level output
	FATAL = &LvlOutput{level: LevelFatal, prefix: "Fatal: ", screenHndl: os.Stderr, screenFlags: 0, logfileHndl: ioutil.Discard, logFlags: LlogfileFlags}

	// Set up all the LvlOutput level details in one array (except discard),
	// the idea that one can control these pretty flexibly (if needed)
	outputters = []*LvlOutput{TRACE, DEBUG, VERBOSE, INFO, NOTE, ISSUE, ERR, FATAL}

	// Set up default/starting logging threshold settings, see SetThreshold()
	// if you wish to change these threshold settings
	screenThreshold = defaultScreenThreshold
	logThreshold    = defaultLogThreshold
	logFileName     string

	// As output is displayed track if last message ended in a newline or not,
	// both to the screen and to the log (as levels may cause output to differ)
	// Note: this is tracked across *all* output levels so if you have done
	// something "interesting" like redirecting to different writers for logfile
	// output (eg: pointing at different log files for different levels) then
	// the below globs don't really work since they treat screen output (all
	// levels as visible in the same "stream" and log output the same way).
	// If you're doing this then you may need to re-work the package a bit,
	// you could track *NewLines for each level independently for example.
	screenNewline  = true
	logfileNewline = true

	// stacktraceOnExit is used to request stack traces,on Fatal*, see
	// SetStacktraceOnExit() below
	stacktraceOnExit = false

	// The below "<..>NameLength" flags help to aligh the output when dumping
	// filenames, line #'s' and function names to a log file in front of the
	// tools normal output.  This is weak (at best), but usually works "ok"
	// for paths, file and func name lengths that tend towards "short".  Note
	// that if you have different log levels to the same output stream using
	// different combos of filename/line# and func name meta-data then your
	// output won't align well (currently), opted not to get too fancy now.

	// ShortFileNameLength is the default "formatting" length for file/line#
	// from runtime.Caller() (just the filename part of the path), right now
	// we'll hope filenames don't usually get longer than 10 chars or so (and
	// there is the :<line#> part of the block which is around 5 chars and
	// then the trailing colon, so we'll go with 16).  If you have longer
	// filenames then you can change this setting so your output alignment
	// improves (or the below settings)
	ShortFileNameLength = 16

	// LongFileNameLength is the full path and filename plus the line # and
	// a trailing colon after that... this is hand-wavy but we'll give it
	// some space for now, adjust as needed for your paths/filenames:
	LongFileNameLength = 55

	// ShortFuncNameLength ties into function names (if those have been added
	// to your output metadata via the Lshortfunc flag), right now it expects
	// method names of around 12 or 13 chars, followed by a colon, adjust as
	// needed for your own method names
	ShortFuncNameLength = 14

	// LongFuncNameLength is the full function name which includes the package
	// name (full path) followed by a dot and then the function name, this may
	// be a bit short for some folks so adjust as needed.
	LongFuncNameLength = 30

	// CallDepth is for runtime.Caller() to identify where a Noteln() or Print()
	// or Issuef() (etc) was called from (so meta-data dumped in "extended"
	// mode gives the correct calling function and line number).  The existing
	// value is correct *but* if you choose to further wrap 'out' methods in
	// some extra method layer (or two) in your own modules then you might
	// want to increase it via this public package global.
	CallDepth = 5

	// DefaultErrCode ties into assigning an error code to all errors so if
	// you aren't using codes (or haven't set them in some err scenarios, which
	// can be normal unless you're applying codes to and wrapping all errors
	// which is unlikely).  Anyhow, the pkg will use this default error code
	// for any error that has no code (mostly internal, if this is an errors
	// code it will not be shown typically)
	DefaultErrCode = 100
)

// levelCheck insures valid log level "values" are provided
func levelCheck(level Level) Level {
	switch {
	case level <= LevelTrace:
		return LevelTrace
	case level >= LevelDiscard:
		return LevelDiscard
	default:
		return level
	}
}

// Threshold returns the current screen or logfile output threshold level
// depending upon which is requested, either out.ForScreen or out.ForLogfile
func Threshold(outputTgt int) Level {
	var threshold Level
	if outputTgt&ForScreen != 0 {
		threshold = screenThreshold
	} else if outputTgt&ForLogfile != 0 {
		threshold = logThreshold
	} else {
		Fatalln("Invalid screen/logfile given for Threshold()")
	}
	return threshold
}

// SetThreshold sets the screen and or logfile output threshold(s) to the given
// level, outputTgt can be set to out.ForScreen, out.ForLogfile or both |'d
// together, level is out.LevelInfo for example (any valid level)
func SetThreshold(level Level, outputTgt int) {
	if outputTgt&ForScreen != 0 {
		screenThreshold = levelCheck(level)
	}
	if outputTgt&ForLogfile != 0 {
		logThreshold = levelCheck(level)
	}
}

// String implements a stringer for the Level type so we can print out string
// representations for the level setting, these names map to the "code" names
// for these settings (not the prefixes for the setting since some levels have
// no output prefix by default).  Client still has full control over "primary"
// out prefix separately from this, see SetPrefix and such.
func (l Level) String() string {
	lvl2String := map[Level]string{
		LevelTrace:   "TRACE",
		LevelDebug:   "DEBUG",
		LevelVerbose: "VERBOSE",
		LevelInfo:    "INFO",
		LevelNote:    "NOTE",
		LevelIssue:   "ISSUE",
		LevelError:   "ERROR",
		LevelFatal:   "FATAL",
		LevelDiscard: "DISCARD",
	}
	l = levelCheck(l)
	return lvl2String[l]
}

// LevelString2Level takes the string representation of a level and turns
// it back into a Level type (integer type/iota)
func LevelString2Level(s string) Level {
	string2Lvl := map[string]Level{
		"TRACE":   LevelTrace,
		"DEBUG":   LevelDebug,
		"VERBOSE": LevelVerbose,
		"INFO":    LevelInfo,
		"NOTE":    LevelNote,
		"ISSUE":   LevelIssue,
		"ERROR":   LevelError,
		"FATAL":   LevelFatal,
		"DISCARD": LevelDiscard,
	}
	if _, ok := string2Lvl[s]; !ok {
		Fatalln("Invalid string level:", s, ", unable to map to Level type")
	}
	return string2Lvl[s]
}

// Prefix returns the current prefix for the given log level
func Prefix(level Level) string {
	level = levelCheck(level)
	if level == LevelDiscard {
		Fatalln("Prefix is not defined for level discard, should never be requested")
	}
	var prefix string
	for _, o := range outputters {
		if o.level == level {
			prefix = o.prefix
			break
		}
	}
	return prefix
}

// SetPrefix sets screen and logfile output prefix to given string, note that
// it is recommended to have a trailing space on the prefix, eg: "Myprefix: "
// unless no prefix is desired then just "" will do
func SetPrefix(level Level, prefix string) {
	level = levelCheck(level)
	if level == LevelDiscard {
		return
	}
	// loop through the levels and reset the prefix of the specified level
	for _, o := range outputters {
		if o.level == level {
			o.mu.Lock()
			defer o.mu.Unlock()
			o.prefix = prefix
		}
	}
}

// Discard disables all screen and/or logfile output, can be done via
// SetThreshold() as well (directly) or via SetWriter() to something
// like ioutil.Discard or bufio io.Writer if you want to capture output.
// Anyhow, this is a quick way to disable output (if outputTgt is not set
// to out.ForScreen or out.ForLogfile or both | together nothing happens)
func Discard(outputTgt int) {
	if outputTgt&ForScreen != 0 {
		SetThreshold(LevelDiscard, ForScreen)
	}
	if outputTgt&ForLogfile != 0 {
		SetThreshold(LevelDiscard, ForLogfile)
	}
}

// Flags gets the screen or logfile output flags (Ldate, Ltime, .. above),
// you must give one or the other (out.ForScreen or out.ForLogfile) only.
func Flags(level Level, outputTgt int) int {
	level = levelCheck(level)
	flags := 0
	for _, o := range outputters {
		o.mu.Lock()
		defer o.mu.Unlock()
		if o.level == level {
			if outputTgt&ForScreen != 0 {
				flags = o.screenFlags
			} else if outputTgt&ForLogfile != 0 {
				flags = o.logFlags
			} else {
				Fatalln("Invalid identification of screen or logfile target for Flags()")
			}
			break
		}
	}
	return (flags)
}

// SetFlags sets the screen and/or logfile output flags (Ldate, Ltime, .. above)
// Note: Right now this sets *every* levels log flags to given value, and one
// can give it out.ForScreen, out.ForLogfile or both or'd together although
// usually one would want to give just one to adjust (screen or logfile)
func SetFlags(level Level, flags int, outputTgt int) {
	for _, o := range outputters {
		o.mu.Lock()
		defer o.mu.Unlock()
		if level == LevelAll || o.level == level {
			if outputTgt&ForScreen != 0 {
				o.screenFlags = flags
			}
			if outputTgt&ForLogfile != 0 {
				o.logFlags = flags
			}
			if level != LevelAll {
				break
			}
		}
	}
}

// Writer gets the screen or logfile output io.Writer for the given log
// level, outputTgt is out.ForScreen or out.ForLogfile depending upon which
// writer you want to grab for the given logging level
func Writer(level Level, outputTgt int) io.Writer {
	level = levelCheck(level)
	writer := ioutil.Discard
	for _, o := range outputters {
		if o.level == level {
			if outputTgt&ForScreen != 0 {
				writer = o.screenHndl
			}
			if outputTgt&ForLogfile != 0 {
				writer = o.logfileHndl
			}
		}
	}
	return (writer)
}

// SetWriter sets the screen and/or logfile output io.Writer for every log
// level to the given writer
func SetWriter(level Level, w io.Writer, outputTgt int) {
	for _, o := range outputters {
		if level == LevelAll || o.level == level {
			o.mu.Lock()
			defer o.mu.Unlock()
			if outputTgt&ForScreen != 0 {
				o.screenHndl = w
			}
			if outputTgt&ForLogfile != 0 {
				o.logfileHndl = w
			}
			if level != LevelAll {
				break
			}
		}
	}
}

// ResetNewline allows one to reset the screen and/or logfile LvlOutput so the
// next bit of output either "thinks" (or doesn't) that the previous output put
// the user on a new line.  If 'val' is true then the next output run through
// this pkg to the given output stream can be prefixed (with timestamps, etc),
// if it is false then no prefix, eg: out.Note("Enter data: ") might produce:
//   Note: enter data: <prompt>
// Which leaves the output stream thinking the last msg had no newline at the
// end of string.  Now, if one's input method reads input with the user hitting
// a newline then the below call can be used to tell the LvlOutput(s) that a
// newline was hit and any fresh output can be prefixed cleanly:
//   out.ResetNewline(true, out.ForScreen|out.ForLogfile)
// Note: for any *output* running through this module this is auto-handled
func ResetNewline(val bool, outputTgt int) {
	if outputTgt&ForScreen != 0 {
		screenNewline = val
	}
	if outputTgt&ForLogfile != 0 {
		logfileNewline = val
	}
}

// LogFileName returns any known log file name (if none returns "")
func LogFileName() string {
	return (logFileName)
}

// SetLogFile uses a log file path (passed in) to result in the log file
// output stream being targeted at this log file (and the log file created).
// Note: as to if anything is actually logged that depends upon the current
// logging level of course (default: LevelDiscard).  Please remember to set
// a log level to turn logging on, eg: SetLogThreshold(LevelInfo)
func SetLogFile(path string) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		Fatalln("Failed to open log file:", path, "Err:", err)
	}
	logFileName = file.Name()
	for _, o := range outputters {
		o.mu.Lock()
		defer o.mu.Unlock()
		o.logfileHndl = file
	}
}

// UseTempLogFile creates a temp file and "points" the fileLogger logger at that
// temp file, the prefix passed in will be the start of the temp file name after
// which Go temp methods will generate the rest of the name, the temp file name
// will be returned as a string, errors will result in Fatalln()
// Note: to finish enabling logging remember to set the logging level to a valid
// level (LevelDiscard is the fileLog default), eg: SetLogThreshold(LevelInfo)
func UseTempLogFile(prefix string) string {
	file, err := ioutil.TempFile(os.TempDir(), prefix)
	if err != nil {
		Fatalln(err)
	}
	logFileName = file.Name()
	for _, o := range outputters {
		o.mu.Lock()
		defer o.mu.Unlock()
		o.logfileHndl = file
	}
	return (logFileName)
}

// Next we head into the <Level>() class methods which don't add newlines
// and simply space separate the options sent to them:

// Trace is the most verbose debug level, space separate opts with no newline
// added and is by default prefixed with "Trace: <date/time> <msg>" for each
// line but you can use flags and remove the timestamp, can also drop the prefix
func Trace(v ...interface{}) {
	TRACE.output(v...)
}

// Debug is meant for basic debugging, space separate opts with no newline added
// and is, by default, prefixed with "Debug: <date/time> <your msg>" for each
// line but you can use flags and remove the timestamp, can also drop the prefix
func Debug(v ...interface{}) {
	DEBUG.output(v...)
}

// Verbose meant for verbose user seen screen output, space separated
// opts printed with no newline added, no output prefix is added by default
func Verbose(v ...interface{}) {
	VERBOSE.output(v...)
}

// Print is meant for "normal" user output, space separated opted
// printed with no newline added, no output prefix is added by default
func Print(v ...interface{}) {
	INFO.output(v...)
}

// Info is the same as Print: meant for "normal" user output, space separated
// opts printed with no newline added and no output prefix added by default
func Info(v ...interface{}) {
	INFO.output(v...)
}

// Note is meant for output of key "note" the user should pay attention to, opts
// space separated and printed with no newline added, "Note: <msg>" prefix is
// also added by default
func Note(v ...interface{}) {
	NOTE.output(v...)
}

// Issue is meant for "normal" user error output, space separated opts
// printed with no newline added, "Issue: <msg>" prefix added by default,
// if you want to exit after the issue is reported see IssueExit()
func Issue(v ...interface{}) {
	ISSUE.output(v...)
}

// IssueExit is meant for "normal" user error output, space separated opts
// printed with no newline added, "Issue: <msg>" prefix added by default,
// the "exit" form of this output routine results in os.Exit() being
// called with the given exitVal (see Issue() if you do not want to exit)
func IssueExit(exitVal int, v ...interface{}) {
	ISSUE.output(v...)
	ISSUE.exit(exitVal)
}

// Error is meant for "unexpected"/system error output, space separated
// opts printed with no newline added, "ERROR: <msg>" prefix added by default,
// if you want to exit after erroring see ErrorExit()
// Note: by "unexpected" these are things like filesystem permissions
// problems, see Issue for more normal user level usage issues
func Error(v ...interface{}) {
	ERR.output(v...)
}

// ErrorExit is meant for "unexpected"/system error output, space separated
// opts printed with no newline added, "ERROR: <msg>" prefix added by default,
// the "exit" form of this output routine results in os.Exit() being called
// with given exitVal (see Error() if you don't want to exit)
// Note: by "unexpected" these are things like filesystem permissions
// problems, see Issue for more normal user level usage issues
func ErrorExit(exitVal int, v ...interface{}) {
	ERR.output(v...)
	ERR.exit(exitVal)
}

// Fatal is meant for "unexpected"/system fatal error output, space separated
// opts printed with no newline added, "FATAL: <msg>" prefix added by default
// and the tool will exit non-zero here
func Fatal(v ...interface{}) {
	FATAL.output(v...)
}

// Next we head into the <Level>ln() class methods which add newlines
// and space separate the options sent to them:

// Traceln is the most verbose debug level, space separate opts with newline
// added and is, by default, prefixed with "Trace: <your output>" for each line
// but you can use flags and remove the timestamp, can also drop the prefix
func Traceln(v ...interface{}) {
	TRACE.outputln(v...)
}

// Debugln is meant for basic debugging, space separate opts with newline added
// and is, by default, prefixed with "Debug: <date/time> <yourmsg>" for each
// line but you can use flags and remove the timestamp, can also drop the prefix
func Debugln(v ...interface{}) {
	DEBUG.outputln(v...)
}

// Verboseln is meant for verbose user seen screen output, space separated
// opts printed with newline added, no output prefix is added by default
func Verboseln(v ...interface{}) {
	VERBOSE.outputln(v...)
}

// Println is the same as Infoln: meant for "normal" user output, space
// separated opts printed with newline added and no output prefix added by
// default
func Println(v ...interface{}) {
	INFO.outputln(v...)
}

// Infoln is the same as Println: meant for "normal" user output, space
// separated opts printed with newline added and no output prefix added by
// default
func Infoln(v ...interface{}) {
	INFO.outputln(v...)
}

// Noteln is meant for output of key items the user should pay attention to,
// opts are space separated and printed with a newline added, "Note: <msg>"
// prefix is also added by default
func Noteln(v ...interface{}) {
	NOTE.outputln(v...)
}

// Issueln is meant for "normal" user error output, space separated
// opts printed with a newline added, "Issue: <msg>" prefix added by default
// Note: by "normal" these are things like unknown codebase name given, etc...
// for unexpected errors use Errorln (eg: file system full, etc).  If you wish
// to exit after your issue is printed please use IssueExitln() instead.
func Issueln(v ...interface{}) {
	ISSUE.outputln(v...)
}

// IssueExitln is meant for "normal" user error output, space separated opts
// printed with a newline added, "Issue: <msg>" prefix added by default,
// the "exit" form of this output routine results in os.Exit() being called
// with the given exitVal.  See Issueln() if you do not want to exit.  This
// routine honors PKG_OUT_NONZERO_EXIT_STACKTRACE env as well as the package
// stacktrace setting via SetStacktraceOnExit(true), only if non-zero exitVal.
func IssueExitln(exitVal int, v ...interface{}) {
	ISSUE.outputln(v...)
	ISSUE.exit(exitVal)
}

// Errorln is meant for "unexpected"/system error output, space separated
// opts printed with a newline added, "ERROR: <msg>" prefix added by default
// Note: by "unexpected" these are things like filesystem permissions problems,
// see Noteln/Issueln for more normal user level notes/usage
func Errorln(v ...interface{}) {
	ERR.outputln(v...)
}

// ErrorExitln is meant for "unexpected"/system error output, space separated
// opts printed with a newline added, "ERROR: <msg>" prefix added by default,
// the "exit" form of this output routine results in os.Exit() being called
// with given exitVal.  If you don't want to exit use Errorln() instead.  This
// routine honors PKG_OUT_NONZERO_EXIT_STACKTRACE env as well as the package
// stacktrace setting via SetStacktraceOnExit(true), only if non-Zero exit val.
// Note: by "unexpected" these are things like filesystem permissions
// problems, see IssueExitln() for more normal user level usage issues
func ErrorExitln(exitVal int, v ...interface{}) {
	ERR.outputln(v...)
	ERR.exit(exitVal)
}

// Fatalln is meant for "unexpected"/system fatal error output, space separated
// opts printed with a newline added, "FATAL: <msg>" prefix added by default
// and the tool will exit non-zero here.  Note that a stacktrace can be added
// for fatal errors, see PKG_OUT_NONZERO_EXIT_STACKTRACE
func Fatalln(v ...interface{}) {
	FATAL.outputln(v...)
}

// Next we head into the <Level>f() class methods which take a standard
// format string for go (see 'godoc fmt' and look at Printf() if needed):

// Tracef is the most verbose debug level, format string followed by args and
// output is, by default, prefixed with "Trace: <date/time> <your msg>" for each
// line but you can use flags and remove the timestamp, can also drop the prefix
func Tracef(format string, v ...interface{}) {
	TRACE.outputf(format, v...)
}

// Debugf is meant for basic debugging, format string followed by args and
// output is by default prefixed with "Debug: <date/time> <your msg>" for each
// line but you can use flags and remove the timestamp, can also drop the prefix
func Debugf(format string, v ...interface{}) {
	DEBUG.outputf(format, v...)
}

// Verbosef is meant for verbose user seen screen output, format string
// followed by args (and no output prefix is added by default)
func Verbosef(format string, v ...interface{}) {
	VERBOSE.outputf(format, v...)
}

// Printf is the same as Infoln: meant for "normal" user output, format string
// followed by args (and no output prefix added by default)
func Printf(format string, v ...interface{}) {
	INFO.outputf(format, v...)
}

// Infof is the same as Printf: meant for "normal" user output, format string
// followed by args (and no output prefix added by default)
func Infof(format string, v ...interface{}) {
	INFO.outputf(format, v...)
}

// Notef is meant for output of key "note" the user should pay attention to,
// format string followed by args, "Note: <yourmsg>" prefixed by default
func Notef(format string, v ...interface{}) {
	NOTE.outputf(format, v...)
}

// Issuef is meant for "normal" user error output, format string followed
// by args, prefix "Issue: <msg>" added by default.  If you want to exit
// after your issue see IssueExitf() instead.
func Issuef(format string, v ...interface{}) {
	ISSUE.outputf(format, v...)
}

// IssueExitf is meant for "normal" user error output, format string followed
// by args, prefix "Issue: <msg>" added by default, the "exit" form of this
// output routine results in os.Exit() being called with the given exitVal.
// If you do not want to exit then see Issuef() instead
func IssueExitf(exitVal int, format string, v ...interface{}) {
	ISSUE.outputf(format, v...)
	ISSUE.exit(exitVal)
}

// Errorf is meant for "unexpected"/system error output, format string
// followed by args, prefix "ERROR: <msg>" added by default
// Note: by "unexpected" these are things like filesystem permissions problems,
// see Notef/Issuef for more normal user level notes/usage
func Errorf(format string, v ...interface{}) {
	ERR.outputf(format, v...)
}

// ErrorExitf is meant for "unexpected"/system error output, format string
// followed by args, prefix "ERROR: <msg>" added by default, the "exit" form
// of this output routine results in os.Exit() being called with given exitVal
func ErrorExitf(exitVal int, format string, v ...interface{}) {
	ERR.outputf(format, v...)
	ERR.exit(exitVal)
}

// Fatalf is meant for "unexpected"/system fatal error output, format string
// followed by args, prefix "FATAL: <msg>" added by default and will exit
// non-zero from the tool (see Go 'log' Fatalf() method)
func Fatalf(format string, v ...interface{}) {
	FATAL.outputf(format, v...)
}

// Exit is meant for terminating without messaging but supporting stack trace
// dump settings and such if non-zero exit.
func Exit(exitVal int) {
	FATAL.exit(exitVal)
}

// SetStacktraceOnExit can be used to flip on stack traces programatically, one
// can also use PKG_OUT_NONZERO_EXIT_STACKTRACE set to "1" as another way, this
// is meant for Fatal[f|ln]() class output/exit
func SetStacktraceOnExit(val bool) {
	stacktraceOnExit = val
}

// getStackTrace will get a stack trace (truncated at 4096 bytes currently)
// if and only if PKG_OUT_NONZERO_EXIT_STACKTRACE is set to "1"
func getStackTrace(exitVal int) string {
	var myStack string
	if exitVal != 0 && (stacktraceOnExit || os.Getenv("PKG_OUT_NONZERO_EXIT_STACKTRACE") == "1") {
		trace, _ := stackTrace(CallDepth)
		myStack = fmt.Sprintf("Stacktrace:\n%s", trace)
	}
	return myStack
}

// InsertPrefix takes a multiline string (potentially) and for each
// line places a string prefix in front of each line, for control
// there are these settings:
//   AlwaysInsert            // Prefix every line, regardless of output history
//   SmartInsert             // Use previous output context to decide on prefix
//   BlankInsert             // Only spaces inserted (same length as prefix)
//   SkipFirstLine           // 1st line in multi-line string has no prefix
func InsertPrefix(s string, prefix string, ctrl int) string {
	if prefix == "" {
		return s
	}
	if ctrl&AlwaysInsert != 0 {
		ctrl = 0 // turn off everything, always means *always*
	}
	pfxLength := len(prefix)
	lines := strings.Split(s, "\n")
	numLines := len(lines)
	newLines := []string{}
	for idx, line := range lines {
		if (idx == numLines-1 && line == "") ||
			(idx == 0 && ctrl&SkipFirstLine != 0) {
			newLines = append(newLines, line)
		} else if ctrl&BlankInsert != 0 {
			format := "%" + fmt.Sprintf("%d", pfxLength) + "s"
			spacePrefix := fmt.Sprintf(format, "")
			newLines = append(newLines, spacePrefix+line)
		} else {
			newLines = append(newLines, prefix+line)
		}
	}
	newstr := strings.Join(newLines, "\n")
	return newstr
}

// output is similar to fmt.Print(), it'll space separate args with no newline
// and output them to the screen and/or log file loggers based on levels
func (o *LvlOutput) output(v ...interface{}) {
	// set up the message to dump
	msg := fmt.Sprint(v...)
	// dump it based on screen and log output levels
	_, err := o.stringOutput(msg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		if os.Getenv("PKG_OUT_NO_EXIT") != "1" {
			os.Exit(-1)
		}
	}
}

// outputln is similar to fmt.Println(), it'll space separate args with no newline
// and output them to the screen and/or log file loggers based on levels
func (o *LvlOutput) outputln(v ...interface{}) {
	// set up the message to dump
	msg := fmt.Sprintln(v...)
	// dump it based on screen and log output levels
	_, err := o.stringOutput(msg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		if os.Getenv("PKG_OUT_NO_EXIT") != "1" {
			os.Exit(-1)
		}
	}
}

// outputf is similar to fmt.Printf(), it takes a format and args and outputs
// the resulting string to the screen and/or log file loggers based on levels
func (o *LvlOutput) outputf(format string, v ...interface{}) {
	// set up the message to dump
	msg := fmt.Sprintf(format, v...)
	// dump it based on screen and log output levels
	_, err := o.stringOutput(msg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		if os.Getenv("PKG_OUT_NO_EXIT") != "1" {
			os.Exit(-1)
		}
	}
}

// exit will use os.Exit() to bail with the given exitVal, if
// that exitVal is non-zero and a stracktrace is set up it will
// dump that stacktrace as well (honoring all log levels and such),
// see getStackTrace() for the env and package settings honored.
func (o *LvlOutput) exit(exitVal int) {
	// get the stacktrace if it's configured
	stacktrace := getStackTrace(exitVal)
	if stacktrace != "" && o.level >= screenThreshold && o.level != LevelDiscard {
		msg, supressOutput := o.doPrefixing(stacktrace, ForScreen, SmartInsert)
		if !supressOutput {
			_, err := o.screenHndl.Write([]byte(msg))
			if err != nil {
				fmt.Fprintf(os.Stderr, "%sError writing stacktrace to screen output handle:\n%+v\n", o.prefix, err)
				if os.Getenv("PKG_OUT_NO_EXIT") != "1" {
					os.Exit(1)
				}
			}
		}
	}
	if stacktrace != "" && o.level >= logThreshold && o.level != LevelDiscard {
		msg, supressOutput := o.doPrefixing(stacktrace, ForLogfile, SmartInsert)
		if !supressOutput {
			o.logfileHndl.Write([]byte(msg))
		}
	}
	if os.Getenv("PKG_OUT_NO_EXIT") != "1" {
		os.Exit(exitVal)
	}
}

// itoa converts an int to fixed-width decimal ASCII.  Give a negative width to
// avoid zero-padding.  Knows the buffer has capacity.  Taken from Go's 'log'
// pkg since we want some of the same formatting.
func itoa(buf *[]byte, i int, wid int) {
	u := uint(i)
	if u == 0 && wid <= 1 {
		*buf = append(*buf, '0')
		return
	}
	// Assemble decimal in reverse order.
	var b [32]byte
	bp := len(b)
	for ; u > 0 || wid > 0; u /= 10 {
		bp--
		wid--
		b[bp] = byte(u%10) + '0'
	}
	*buf = append(*buf, b[bp:]...)
}

// getFlagString takes the time the output func was called and tries
// to construct a string to put in the log file (uses the flags settings
// to decide what metadata to print, ie: one can "or" together different
// flags to identify what should be dumped, like the Go 'log' package but
// more flags are available, see top of file)
func getFlagString(buf *[]byte, flags int, level Level, funcName string, file string, line int, t time.Time) string {
	if flags&Lpid != 0 {
		pid := os.Getpid()
		*buf = append(*buf, '[')
		itoa(buf, pid, 1)
		*buf = append(*buf, "] "...)
	}
	if flags&Llevel != 0 {
		lvl := fmt.Sprintf("%-8s", level)
		*buf = append(*buf, lvl...)
	}
	if flags&(Ldate|Ltime|Lmicroseconds) != 0 {
		if flags&Ldate != 0 {
			year, month, day := t.Date()
			itoa(buf, year, 4)
			*buf = append(*buf, '/')
			itoa(buf, int(month), 2)
			*buf = append(*buf, '/')
			itoa(buf, day, 2)
			*buf = append(*buf, ' ')
		}
		if flags&(Ltime|Lmicroseconds) != 0 {
			hour, min, sec := t.Clock()
			itoa(buf, hour, 2)
			*buf = append(*buf, ':')
			itoa(buf, min, 2)
			*buf = append(*buf, ':')
			itoa(buf, sec, 2)
			if flags&Lmicroseconds != 0 {
				*buf = append(*buf, '.')
				itoa(buf, t.Nanosecond()/1e3, 6)
			}
			*buf = append(*buf, ' ')
		}
	}
	if flags&(Lshortfile|Llongfile) != 0 {
		formatLen := LongFileNameLength
		if flags&Lshortfile != 0 {
			formatLen = ShortFileNameLength
			short := file
			for i := len(file) - 1; i > 0; i-- {
				if file[i] == '/' {
					short = file[i+1:]
					break
				}
			}
			file = short
		}
		var tmpbuf []byte
		tmpslice := &tmpbuf
		*tmpslice = append(*tmpslice, file...)
		*tmpslice = append(*tmpslice, ':')
		itoa(tmpslice, line, -1)
		if flags&Lshortfunc != 0 {
			formatLen = formatLen + ShortFuncNameLength
			parts := strings.Split(funcName, ".")
			var justFunc string
			if len(parts) > 1 {
				justFunc = parts[len(parts)-1]
			} else {
				justFunc = "???"
			}
			*tmpslice = append(*tmpslice, ':')
			*tmpslice = append(*tmpslice, justFunc...)
		} else if flags&Llongfunc != 0 {
			formatLen = formatLen + LongFuncNameLength
			*tmpslice = append(*tmpslice, ':')
			*tmpslice = append(*tmpslice, funcName...)
		} else {
			*tmpslice = append(*tmpslice, ' ')
		}

		// Note that this length stuff is weak, if you have long filenames,
		// long func names or long paths to func's it won't do much good as
		// it's currently written (or if you have different flags across
		// different log levels... but if consistent then it can help a bit)
		formatStr := "%-" + fmt.Sprintf("%d", formatLen) + "s: "
		str := fmt.Sprintf(formatStr, string(*tmpslice))
		*buf = append(*buf, str...)
	}
	return fmt.Sprintf("%s", *buf)
}

// determineFlags takes a set of flags defined in an env var (string) that
// can be comma separated and turns them into a real flags store type (int) with
// the desired settings, allows easy dynamic tweaking or addition of flags in
// screen output for instance
func determineFlags(flagStr string) int {
	flagStrs := strings.Split(flagStr, ",")
	flags := 0
	for _, currFlag := range flagStrs {
		switch currFlag {
		case "debug":
			flags |= Llevel | Ltime | Lmicroseconds | Lshortfile | Lshortfunc
		case "all":
			flags |= Lpid | Llevel | Ldate | Ltime | Lmicroseconds | Lshortfile | Lshortfunc
		case "longall":
			flags |= Lpid | Llevel | Ldate | Ltime | Lmicroseconds | Llongfile | Llongfunc
		case "pid":
			flags |= Lpid
		case "level":
			flags |= Llevel
		case "date":
			flags |= Ldate
		case "time":
			flags |= Ltime
		case "micro", "microseconds":
			flags |= Lmicroseconds
		case "file", "shortfile":
			flags |= Lshortfile
		case "longfile":
			flags |= Llongfile
		case "func", "shortfunc":
			flags |= Lshortfunc
		case "longfunc":
			flags |= Llongfunc
		case "off":
			flags = 0
			break
		default:
		}
	}
	return flags
}

// insertFlagMetadata basically checks to see what flags are set for
// the current screen or logfile output and inserts the meta-data in
// front of the string, see InsertPrefix for ctrl description, outputTgt
// here is either ForScreen or ForLogfile (constants) for output.  Note that
// it will also return a boolean to indicate if the output should be supressed
// or not (typically not but one can filter debug/trace output and if one has
// set PKG_OUT_DEBUG_SCOPE, see env var elsewhere in this pkg for doc)
func (o *LvlOutput) insertFlagMetadata(s string, outputTgt int, ctrl int) (string, bool) {
	now := time.Now() // do this before Caller below, can take some time
	var file string
	var funcName string
	var line int
	var flags int
	var supressOutput bool
	var level Level
	o.mu.Lock()
	defer o.mu.Unlock()
	// if printing to the screen target use those flags, else use logfile flags
	if outputTgt&ForScreen != 0 {
		if str := os.Getenv("PKG_OUT_SCREEN_FLAGS"); str != "" {
			flags = determineFlags(str)
		} else {
			flags = o.screenFlags
		}
		level = o.level
	} else if outputTgt&ForLogfile != 0 {
		if str := os.Getenv("PKG_OUT_LOGFILE_FLAGS"); str != "" {
			flags = determineFlags(str)
		} else {
			flags = o.logFlags
		}
		level = o.level
	} else {
		Fatalln("Invalid target passed to insertFlagMetadata():", outputTgt)
	}
	supressOutput = false
	if flags&(Lshortfile|Llongfile|Lshortfunc|Llongfunc) != 0 ||
		os.Getenv("PKG_OUT_DEBUG_SCOPE") != "" {
		// Caller() can take a little while so unlock the mutex
		o.mu.Unlock()
		var ok bool
		var pc uintptr
		pc, file, line, ok = runtime.Caller(CallDepth)
		if !ok {
			file = "???"
			line = 0
			funcName = "???"
		} else {
			f := runtime.FuncForPC(pc)
			if f == nil {
				funcName = "???"
			} else {
				funcName = f.Name()
			}
		}
		o.mu.Lock()
		// if the user has restricted debugging output to specific packages
		// or methods (funcname might be "github.com/dvln/out.MethodName")
		// then we'll supress all debug output outside of the desired scope and
		// only show those packages or methods of interest... simple substring
		// match is done currently
		if debugScope := os.Getenv("PKG_OUT_DEBUG_SCOPE"); funcName != "???" && debugScope != "" && (o.level == LevelDebug || o.level == LevelTrace) {
			scopeParts := strings.Split(debugScope, ",")
			supressOutput = true
			for _, scopePart := range scopeParts {
				if strings.Contains(funcName, scopePart) {
					supressOutput = false
					break
				}
			}
		}
	}
	o.buf = o.buf[:0]
	leader := getFlagString(&o.buf, flags, level, funcName, file, line, now)
	if leader == "" {
		return s, supressOutput
	}
	s = InsertPrefix(s, leader, ctrl)
	return s, supressOutput
}

// doPrefixing takes the users output string and decides how to prefix
// the users message based on the log level and any associated prefix,
// eg: "Debug: ", as well as any flag settings that could add date/time
// and information on the calling Go file and line# and such.
//
// An example of what prefixing means might be useful here, if our code has:
//   [13:]  out.Noteln("This is a test\n", "and only a test\n")
//   [14:]  out.Noteln("that I am showing to ")
//   [15:]  out.Notef("%s\n", getUserName())
//   [16:]  out.Noteln("...")
// It would result in output like so to the screen (typically, flags to adjust):
//   Note: This is a test
//   Note: and only a test
//   Note: that I am showing to John
// Aside: other levels like Debug and Trace add in date/time to screen output
// Log file entry and formatting for the same code if logging is active:
//   <date/time> myfile.go:13: Note: This is a test
//   <date/time> myfile.go:13: Note: and only a test
//   <date/time> myfile.go:14: Note: that I am showing to John
// The only thing we "lose" here potentially is that the line that prints
// the username isn't be prefixed to keep the output clean (no line #15 details)
// hence we don't have a date/timestamp for that "part" of the output and that
// could cause someone to think it was line 14 that was slow if the next entry
// was 20 minutes later (eg: the myfile.go line 16 print statement).  There is
// a mode to turn off smart flags prefixing so one can see such "invisible"
// or missing timestamps on the same line... to do that one would set the env
// PKG_OUT_SMART_FLAGS_PREFIX to "off".  For screen output default settings
// this changes nothing (flags are off for regular/note/issue/err output).
// However, the log file entry differs as we can see in the 3rd line, we
// now see the timestamp and file info for both parts of that line:
//   <date/time> myfile.go:13: Note: This is a test
//   <date/time> myfile.go:13: Note: and only a test
//   <date/time> myfile.go:14: Note: that I am showing to <date/time> myfile:15: John
// Obviously makes the output uglier but might be of use now and then.
//
// One more note, if a stack trace is added on a Fatal error (if turned on)
// then we force add a newline if the fatal doesn't have one and dump the
// stack trace with 'BlankInsert' so the stack trace is associated with
// that fatal print, eg:
//   os.Setenv("PKG_OUT_NONZERO_EXIT_STACKTRACE", "1")
//   out.Fatal("Severe error, giving up\n")    [use better errors of course]
// Screen output:
//   FATAL: Severe error, giving up
//   FATAL: <multiline stacktrace here>
// Log file entry:
//   <date/time> myfile.go:37: FATAL: Severe error, giving up
//   <date/time> myfile.go:37: FATAL: <multiline stacktrace here>
// The goal being readability of the screen and logfile output while conveying
// information about date/time and source of the fatal error and such
func (o *LvlOutput) doPrefixing(s string, outputTgt int, ctrl int) (string, bool) {
	// where we check out if we previously had no newline and if so the
	// first line (if multiline) will not have the prefix, see example
	// in function header around username
	var onNewline bool
	if outputTgt&ForScreen != 0 {
		onNewline = screenNewline
	} else if outputTgt&ForLogfile != 0 {
		onNewline = logfileNewline
	} else {
		Fatalln("Invalid target for output given in doPrefixing():", outputTgt)
	}
	if !onNewline && ctrl&SmartInsert != 0 {
		ctrl = ctrl | SkipFirstLine
	}
	s = InsertPrefix(s, o.prefix, ctrl)

	if os.Getenv("PKG_OUT_SMART_FLAGS_PREFIX") == "off" {
		ctrl = AlwaysInsert // forcibly add prefix without smarts
	}
	// now set up metadata prefix (eg: timestamp), if any, same as above
	// it has the brains to not add in a prefix if not needed or wanted
	var supressOutput bool
	s, supressOutput = o.insertFlagMetadata(s, outputTgt, ctrl)
	return s, supressOutput
}

// stringOutput uses existing screen and log levels to decide what, if
// anything, is printed to the screen and/or log file Writer(s) based on
// current screen and log output thresholds, flags and stack trace settings.
// It returns the length of output written (to *both* screen and logfile targets
// if it succeeds... and note that the length will include additional meta-data
// that the user has requested be added) and an error if one occurred.
func (o *LvlOutput) stringOutput(s string) (int, error) {
	// print to the screen output writer first...
	var stacktrace string
	if o.level == LevelFatal {
		stacktrace = getStackTrace(-1)
	}
	var err error
	var n int
	var screenLength int
	var logfileLength int
	if o.level >= screenThreshold && o.level != LevelDiscard {
		pfxScreenStr, supressOutput := o.doPrefixing(s, ForScreen, SmartInsert)
		if !supressOutput && s != "" {
			//FIXME: erik: now that we have error codes (always) we need to
			//    set up an optional formatter interface which dvln can
			//    register... for JSON.  Should allow for non-terminal and
			//    terminal formatters for both screen and logfile (terminal
			//    only applies to ISSUE w/exit, ERROR w/exit or FATAL).  Using
			//    that the terminal ones would convert the msg to a JSON error
			//    (with msg, level and msg code), return that and it would be
			//    dumped here.  For non-terminal the formatter would do this:
			//    a) use an 'api' pkg API to store the warning in JSON w/msg,
			//       level, msg code (if DetailedError the msg would be
			//       potentially multi-line, might have stacktrace as well
			//       if that is active for msgs)
			//    b) indicate to this method NOT to dump any output
			//
			//    Question: for the codes, should error and regular msg
			//       codes come in the same context so DetailedError could
			//       become DetailedMsg instead (and be used for errs or Msgs,
			//       should Msgs allow "nesting" like errors with diff codes
			//       and the same logic to use the most outer code?)
			//
			//     Consider: stack trace handling should be smarter if it's
			//       a DetailedError, use the same logic used by Error() on
			//       that type to get the "lowest" stack trace possible which
			//       will already be in the messages (and if not available or
			//       regular errors then we'll use existing functionality).
			//       Also need to consider stack traces on exit, non-exit...
			//       should we support stack traces:
			//       - only on non-zero exit issues/errors/fatal
			//       - on any issues/errors/fatal
			//       - both of the above, configurable (likely), default is ???
			//       - on message at all (probably not, too costly)

			n, err = o.screenHndl.Write([]byte(pfxScreenStr))
			screenLength += n
			if err != nil {
				myerr := fmt.Errorf("%sError writing to screen output handler:\n%+v\noutput:\n%s\n", o.prefix, err, s)
				return screenLength, myerr
			}
			if s[len(s)-1] == 0x0A { // if last char is a newline..
				screenNewline = true
			} else {
				screenNewline = false
			}
		}
		if o.level == LevelFatal {
			if !screenNewline {
				// ignore errors, just quick "prettyup" attempt:
				n, err = o.screenHndl.Write([]byte("\n"))
				screenLength += n
				if err != nil {
					myerr := fmt.Errorf("%sError writing newline to screen output handler:\n%+v\n", o.prefix, err)
					return screenLength, myerr
				}
			}
			pfxScreenStr, _ = o.doPrefixing(stacktrace, ForScreen, SmartInsert)
			// don't need to check supressOutput, possible for debug/trace only
			n, err = o.screenHndl.Write([]byte(pfxScreenStr))
			screenLength += n
			if err != nil {
				myerr := fmt.Errorf("%sError writing stacktrace to screen output handle:\n%+v\n", o.prefix, err)
				return screenLength, myerr
			}
		}
	}

	// print to the log file writer next
	if o.level >= logThreshold && o.level != LevelDiscard {
		pfxLogfileStr, supressOutput := o.doPrefixing(s, ForLogfile, SmartInsert)
		if !supressOutput && s != "" {
			n, err = o.logfileHndl.Write([]byte(pfxLogfileStr))
			logfileLength += n
			if err != nil {
				myerr := fmt.Errorf("%sError writing to logfile output handler:\n%+v\noutput:\n%s\n", o.prefix, err, s)
				return logfileLength, myerr
			}
			if s[len(s)-1] == 0x0A {
				logfileNewline = true
			} else {
				logfileNewline = false
			}
		}
		if o.level == LevelFatal {
			if !logfileNewline {
				o.logfileHndl.Write([]byte("\n"))
				logfileLength += n
				if err != nil {
					myerr := fmt.Errorf("%sError writing newline to screen output handler:\n%+v\n", o.prefix, err)
					return logfileLength, myerr
				}
			}
			pfxLogfileStr, _ = o.doPrefixing(stacktrace, ForLogfile, SmartInsert)
			n, err = o.logfileHndl.Write([]byte(pfxLogfileStr))
			logfileLength += n
			if err != nil {
				myerr := fmt.Errorf("%sError writing stacktrace to logfile output handle:\n%+v\n", o.prefix, err)
				return logfileLength, myerr
			}
		}
	}
	// if we're fatal erroring then we need to exit unless overrides in play,
	// this env var should be used for test suites only really...
	if o.level == LevelFatal &&
		os.Getenv("PKG_OUT_NO_EXIT") != "1" {
		os.Exit(-1)
	}
	// if all good return all the bytes we wrote to *both* targets and nil err
	return logfileLength + screenLength, nil
}

// GetWriter will return an io.Writer compatible structure for the desired
// output level.  It's a bit cheesy but does the trick if you want an
// io.Writer at a given level.  Typically one would not use this and
// would instead just pass in out.TRACE, out.DEBUG, out.VERBOSE, out.INFO,
// out.NOTE, out.ISSUE, out.ERROR or out.FATAL directly as the io.Writer
// to write at a given output level (but if you have a Level type and
// want to get the associated io.Writer you can use this method)
func GetWriter(l Level) *LvlOutput {
	var writeLevel *LvlOutput
	l = levelCheck(l)
	switch l {
	case LevelTrace:
		writeLevel = TRACE
	case LevelDebug:
		writeLevel = DEBUG
	case LevelVerbose:
		writeLevel = VERBOSE
	case LevelInfo:
		writeLevel = INFO
	case LevelNote:
		writeLevel = NOTE
	case LevelIssue:
		writeLevel = ISSUE
	case LevelError:
		writeLevel = ERR
	case LevelFatal:
		writeLevel = FATAL
	default:
		writeLevel = INFO
	}
	return writeLevel
}

// Write implements an io.Writer interface for any of the available output
// levels.  Use GetWriter() above to grab a *LvlOutput structure for the
// desired output level... so, if you want the "standard" info (print) output
// level then one might do this:
//   infoWriter := out.GetWriter(out.LevelInfo)
//   fmt.Fprintf(infoWriter, "%s\n", stringVar)
// The above example would print to the screen and any logfile that was set up
// just like the Info[ln|f]() (or the Print[ln|f]()) routine would.  Keep in
// mind that if a logfile has been activated this io.Writer will behave somewhat
// like an io.MultiWriter (writing to multiple target handles potentially, the
// difference being that here the different target handles can be augmented with
// independently controlled levels of additional meta-data, independent output
// levels for each target handle, etc (and one could combine this io.Writer with
// additional writers itself via io.MultiWriter even, crazy fun)
func (o *LvlOutput) Write(p []byte) (n int, err error) {
	return o.stringOutput(string(p))
}

// GetMessage returns the error string without stack trace information, note
// that this will recurse across all nested errors whereas the use of something
// like "detErr.GetMessage()" would only return the message *in* that one error
// even if it was part of a set of nested/inner errors.
func GetMessage(err interface{}) string {
	switch e := err.(type) {
	case DetailedError:
		detErr := DetailedError(e)
		ret := []string{}
		for detErr != nil {
			ret = append(ret, detErr.GetMessage())
			i := detErr.GetInner()
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
		return "Passed a non-error to GetMessage"
	}
}

// GetCode returns the errors code (if no code, ie: code=0, then the "default"
// error code (100, as set in DefaultErrCode) will be returned.  This routine
// will recurse across all nested/inner errors, the basics:
// a) the "most outer" code that is not 0 or DefaultErrCode (if 3 nested errors
// and the middle is set to 209 and the rest aren't set, ie: 0 or DefaultErrCode
// then 209 will be the err code returned)
// b) if nothing is set then return the DefaultErrCode (typically 100)
// This is different than "detErr.GetCode()" as that will get whatever code
// is set for that specific error only (will not recurse inner errors/etc)
func GetCode(err interface{}) int {
	switch e := err.(type) {
	case DetailedError:
		detErr := DetailedError(e)
		code := 0
		for detErr != nil {
			code = detErr.GetCode()
			if code != 0 && code != DefaultErrCode {
				break
			}
			i := detErr.GetInner()
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
			code = DefaultErrCode
		}
		return code
	default:
		return DefaultErrCode
	}
}

// Error returns a string with all available error information, including inner
// errors that are wrapped by this errors.
func (e *BaseError) Error() string {
	return DefaultError(e)
}

// GetMessage returns the error message without the stack trace.  Note that
// this will not recurse inner/nested errors at all, see "GetMessage(someErr)"
// for that functionality (vs. this being called via "detErr.GetMessage()")
func (e *BaseError) GetMessage() string {
	return e.Msg
}

// GetStack returns the stack trace without the error message.
func (e *BaseError) GetStack() string {
	return e.Stack
}

// GetCode returns the code, if any, available in the given error... note that
// this will not recurse inner/nested errors at all, see "GetCode(someErr)" for
// that functionality (vs. this being called via "detErr.GetCode()")
func (e *BaseError) GetCode() int {
	if e.Code == 0 {
		e.Code = DefaultErrCode
	}
	return e.Code
}

// GetContext returns the stack trace's context.
func (e *BaseError) GetContext() string {
	return e.Context
}

// GetInner returns the wrapped error, if there is one.
func (e *BaseError) GetInner() error {
	return e.inner
}

// Err returns a new BaseError initialized with the given message and
// the current stack trace.
func Err(msg string, code ...int) DetailedError {
	stack, context := StackTrace()
	errNum := 0
	if code != nil {
		errNum = code[0]
	}
	return &BaseError{
		Msg:     msg,
		Code:    errNum,
		Stack:   stack,
		Context: context,
	}
}

// Errf is the same as Err, but with fmt.Printf-style params and error
// code # required
func Errf(format string, code int, args ...interface{}) DetailedError {
	stack, context := StackTrace()
	return &BaseError{
		Msg:     fmt.Sprintf(format, args...),
		Code:    code,
		Stack:   stack,
		Context: context,
	}
}

// WrapErr wraps another error in a new BaseError.
func WrapErr(err error, msg string, code ...int) DetailedError {
	stack, context := StackTrace()
	errNum := 0
	if code != nil {
		errNum = code[0]
	}
	return &BaseError{
		Msg:     msg,
		Code:    errNum,
		Stack:   stack,
		Context: context,
		inner:   err,
	}
}

// WrapErrf is the same as WrapErr, but with fmt.Printf-style parameters and
// a required error code #
func WrapErrf(err error, code int, format string, args ...interface{}) DetailedError {
	stack, context := StackTrace()
	return &BaseError{
		Msg:     fmt.Sprintf(format, args...),
		Code:    code,
		Stack:   stack,
		Context: context,
		inner:   err,
	}
}

// DefaultError is a default implementation of the Error method of the detailed
// error interface, see "(DetailedError) Error()" in this pkg.
func DefaultError(e DetailedError) string {
	// Find the "original" stack trace, which is probably the most helpful for
	// debugging.
	errLines := make([]string, 1)
	var origStack string
	code := GetCode(e)
	errLines[0] = fmt.Sprintf("Error %d:", code)
	fillErrorInfo(e, &errLines, &origStack)
	errLines = append(errLines, "")
	errLines = append(errLines, "Stacktrace:")
	errLines = append(errLines, origStack)
	return strings.Join(errLines, "\n")
}

// fillErrorInfo fills errLines with all error messages, and origStack with the
// inner-most stack.
func fillErrorInfo(err error, errLines *[]string, origStack *string) {
	if err == nil {
		return
	}

	derr, ok := err.(DetailedError)
	if ok {
		*errLines = append(*errLines, derr.GetMessage())
		*origStack = derr.GetStack()
		fillErrorInfo(derr.GetInner(), errLines, origStack)
	} else {
		*errLines = append(*errLines, err.Error())
	}
}

// stackTrace returns a copy of the error with the stack trace field populated
// and any other shared initialization; skips 'skip' levels of the stack trace.
// The cleaned up "current" stack trace is returned as is anything that might
// be visible after it as 'context'.  This was borrowed from Dropbox's open
// 'errors' package and frankly I'm not clear as to if 'context' is ever
// non-empty (based on stack traces I've seen and the parsing below I think
// it will always be empty but I might be missing something)
// NOTE: This can panic if any error (eg: runtime stack trace gathering issue)
func stackTrace(skip int) (current, context string) {
	// grow buf until it's large enough to store entire stack trace
	buf := make([]byte, 128)
	for {
		n := runtime.Stack(buf, false)
		if n < len(buf) {
			buf = buf[:n]
			break
		}
		buf = make([]byte, len(buf)*2)
	}

	// Returns the index of the first occurrence of '\n' in the buffer 'b'
	// starting with index 'start'.
	//
	// In case no occurrence of '\n' is found, it returns len(b). This
	// simplifies the logic on the calling sites.
	indexNewline := func(b []byte, start int) int {
		if start >= len(b) {
			return len(b)
		}
		searchBuf := b[start:]
		index := bytes.IndexByte(searchBuf, '\n')
		if index == -1 {
			return len(b)
		}
		return (start + index)
	}

	// Strip initial levels of stack trace, but keep header line that
	// identifies the current goroutine.
	var strippedBuf bytes.Buffer
	index := indexNewline(buf, 0)
	if index != -1 {
		strippedBuf.Write(buf[:index])
	}

	// Skip lines.
	for i := 0; i < skip; i++ {
		index = indexNewline(buf, index+1)
		index = indexNewline(buf, index+1)
	}

	isDone := false
	startIndex := index
	lastIndex := index
	for !isDone {
		index = indexNewline(buf, index+1)
		if (index - lastIndex) <= 1 {
			isDone = true
		} else {
			lastIndex = index
		}
	}
	strippedBuf.Write(buf[startIndex:index])
	return strippedBuf.String(), string(buf[index:])
}

// StackTrace returns the current stack trace string.  NOTE: the stack creation
// code is excluded from the stack trace.
func StackTrace() (current, context string) {
	return stackTrace(3)
}

// unwrapError returns a wrapped error or nil if there is none.
func unwrapError(ierr error) (nerr error) {
	// Internal errors have a well defined bit of context.
	if detErr, ok := ierr.(DetailedError); ok {
		return detErr.GetInner()
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
			currCode := detErr.GetCode()
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
// that uses a matching code (assuming non-0 and not set to the DefaultErrCode
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
			if val != 0 && val != DefaultErrCode {
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
