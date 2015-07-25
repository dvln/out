out
===

A package for "leveled", easily "mirrored" (eg: logfile/buffer) and optionally
"augmented" (eg: pid, date/timestamp, Go file/func/line# data, etc) CLI output
stream management.  Designed for easy drop-in 'fmt' replacement with no setup
required for all your output.  Eg: fmt.Println, fmt.Print and fmt.Printf would
become out.Println, out.Print and out.Printf but with ability to also output
at various other levels like out.Verboseln, out.Debugf, out.Fatal, out.Issue,
out.Noteln, out.Printf, etc.  One can also flexibly adjust screen and log file
io.Writer settings to send output anywhere (or shut it off).

Note that for debug and trace output levels (ie: trace = verbose debug) one can also
control which of your Go package(s) or function(s) will have debug info displayed.
By default all debug data is printed (if your output thresholds indicate to print
it of course) but using the debug scope one can dynamically restrict any tool using
this package to only desired code areas when that is helpful.  See the env vars
section at the bottom for details.

Example fake CLI tool that takes a JSON config file (setting there says to turn
on "record", ie: a log file, with default log file output flags) and what the
screen and logfile data might look like by default:

```text
% cat ~/.mytool/cfg.json 
{ "record" : "~/path/to/logfile.txt"
}
% mytool get --flag1=test --flag2=julius --choke=x
Look up flag1
Look up flag2
We have flag1 test, flag2 julius
Error: Problem x indicated
% cat ~/path/to/logfile.txt 
[616] INFO    2015/07/25 01:05:01.886736 get.go:75:get                 : Look up flag1
[616] INFO    2015/07/25 01:05:01.886882 get.go:78:get                 : Look up flag2
[616] INFO    2015/07/25 01:05:01.886913 summary.go:239:printResult    : We have flag1 test, flag2 julius
[616] ERROR   2015/07/25 01:05:01.887011 problem.go:32:dumpErr         : Error: Problem x indicated
[616] ERROR   2015/07/25 01:05:01.887011 problem.go:32:dumpErr         : Error:
[616] ERROR   2015/07/25 01:05:01.887011 problem.go:32:dumpErr         : Error: Stack Trace: goroutine 1 [running]:
[616] ERROR   2015/07/25 01:05:01.887011 problem.go:32:dumpErr         : Error: ...[stack trace details]...
% 

```

Screen output is clean, in the logfile one sees details about each line of
screen output such as the pid, the general output level (print=info), the
date/time, the go file and line # for each output call as well as the function
name... along with the output formatted to align fairly well so it's easy to
compare the log output to the screen output (and see what the user sees).  If
long filenames were desired with pkg name info that is available as well, same
with longer function name information.  If configured trace/debug could be
added to the logfile output stream while not showing them on the users screen
stream.  Independent control can be powerful.

Also builtin is an implementation of "detailed" errors.  These are optional
and can be used for new errors or to wrap Go errors (or themselves)... so they
are "nested" and one can stack errors as one backs out of a function/method
hierarchy.  All stacked errors will dump the most recent error with any earlier
errors shown under it and the stack trace used will be from the oldest error that
can be found (where the error chain started, gives the most detailed stack trace).
This can greatly help in troubleshooting.  Detailed errors implement the standard
error interface and, themselves, are interfaces that one can augment.  One can also
check Go stdlib or pkg related standard Go error values for equivalence against a
detailed error (and if that regular Go error was wrapped it will match).  One can
optionally use error codes (numbers) in these detailed errors as well (and match
on error code as well).  If using detailed errors the Issue, Error, and Fatal
output levels will take advantage of them and show the best stack trace and
combined messages, incorporating any error codes used as well (if error codes
are used then "Error: " prefixes become "Error #1004: " automatically, same for
Issue and Fatal level output).  If not used, no problem.

The goal is flexible control of output streams and how they can be marked
up, dynamically redirected, augmented with meta-data to help with support
or troubleshooting, etc.  If something fancier is needed one can dynamically
plug in your own output formatter with this package to override or augment
the built-in output prefixes and flags.  For example, my tool has a text and
JSON output mode.  I use this feature when in JSON mode to control error
Issues, Errors or Fatals so they tie into the JSON infrastructure and return
valid JSON containing errors or warnings (vs simple text messages).  These
formatters can completely suppress the "native" 'out' package output for
either the screen or log file or both output streams (eg: my JSON package
does this for warnings, pushing them into the JSON structure I'll dump at
the end so I "stash" them in the JSON output structure and tell the 'out'
package not to print them or log them since the final JSON output dump will
include them).

Summary:

1. Ready for basic CLI screen output with levels out of the box, no setup
2. Easy drop-in replacement for fmt.Print[f|ln](), plus level specific func's
3. Trivial to "turn on" logging (output mirroring) to a temp or named log file
4. Independent io.Writer control over the two output streams (eg: screen/logfile)
5. Independent output level thresholds for each target (eg: screen and logfile)
6. Independent control over flags (eg: augment log file with date/time, file/line#, etc)
7. Clean alignment and handling for multi-line strings or strings w/no newlines
8. Ability to limit debug/trace output to specific pkg(s) or function(s)
9. Ability to easily add stack trace on issues/errors/fatal's (dying/non-zero or not)
10. Goal is to be "safe" for concurrent use (if you see problems please open a git issue)
11. Support for plugin formatters to roll your own format (or support other output mechanisms)
12. Optional: "detailed" errors type adds stack from orig error instance, wrapping of errors

Note: more examples on the last couple of options will be forthcoming.

Thanks much to the Go community for ideas, open packages, etc to form some of these
ideas (particularly the Go authors, spf13 and the Dropbox folks).

# Usage

## Basic usage:
Put calls throughout your code based on desired "levels" of messaging.
Simply run calls to the output functions:

 * out.Trace\[f|ln\](...)  
 * out.Debug\[f|ln\](...)
 * out.Verbose\[f|ln\](...)
 * out.Info\[f|ln\](...) or out.Print\[f|ln\](...)              (identical)
 * out.Note\[f|ln\](...)
 * out.Issue\[f|ln\](...) or out.IssueExit\[f|ln\](exitVal, ..) (2nd form exits)
 * out.Error\[f|ln\](...) or out.ErrorExit\[f|ln\](exitVal, ..) (2nd form exits)
 * out.Fatal\[f|ln\](...)                                       (always exits)

Each of these map to two io.Writers, one "defaulting" for the screen and the 
other usually targeted towards log file output (default is to discard log file
output until it is configured, see below).  One can, of course, redirect either
or both of these output streams anywhere via io.Writers (or multi-Writers).
One can also send output to the two output streams via an io.Writer at the
desired level.  There are a couple ways to get a writer but the easiest way
is direct: out.TRACE, out.DEBUG, out.VERBOSE, out.INFO, out.NOTE, out.ISSUE,
out.ERROR, out.FATAL, eg:

 * fmt.Fprintf(out.DEBUG, "%s", someDebugString)    (same as out.Debug(..))
 * fmt.Fprintf(out.INFO, "%s", someString)          (same as out.Print|out.Info)
 * ...

One can also access the writers via API but not as easy as the above:

 * fmt.Fprintf(out.GetWriter(out.LevelInfo), "%s", someString)
 * fmt.Fprintf(out.GetWriter(out.LevelDebug), "%s", someDebugString)
 * ...

One can additionally override the screen and logfile io.Writers for any
output level via SetWriter(), see below.  Keep in mind that you may need
to adjust the output thresholds so you actually see output once a writer
is set, see SetThreshold() below.

An example of standard usage:

```go
    import (
        "github.com/dvln/out"
    )

    ...
    // Day to day normal output
    out.Printf("some info: %s\n", infostr)

    // Less important output not shown unless in verbose mode
    out.Verbosef("data was pulled from src: %s\n", source)

    ...
    if err != nil {
        // Something like an issue to take note of but recoverable,
        // here output is:
        //   Issue: <error>
        //   Issue: Recovering and continuing with the process ..
        out.Issue(err, "\nRecovering and continuing ")
        out.Issueln("with the process ..")
    }
    ...
    if err != nil {
        // Maybe this is a more severe unexpected error, but recoverable
        out.Errorf("File read failure (file: %s), ignoring: %s\n", file, err)
    }
    ...
    if err != nil {
        // This is a fatal error that we want a stack trace for our screen
        // output for (could use ForLogfile or ForBoth as well)... and one
        // can indicate non-zero exits (as below), any error exit via the
        // "StackTraceErrorExit" setting or via StackTraceAllIssues one can
        // cause a trace for any type of warning/error (Issue/Error/Fatal):
        // Note: the env PKG_OUT_STACK_TRACE_CONFIG can be used as alternative
        //       dynamic way to kick on stack traces without using the API
        //       call below for SetStackTraceConfig().. eg:  set the env
        //       var to "screen,allissues" or "both,nonzeroerrorexit" or
        //       perhaps "logfile,allissues" to kick it on).  For the API:
		out.SetStackTraceConfig(out.ForScreen|out.StackTraceNonZeroErrorExit)
        out.Fatalln(err)
    }

```

There are 8 output levels (perhaps too many for most folks) but one can, of
course, just use those that a given product needs.  There is no need to use
levels you do not want.

Quick note: some packages like spf13's 'viper' have been ported to 'out' in
my fork of these packages, see:"github.com/dvln/viper" for example.

The default settings add default "prefixes" on some of the messages, the
defaults for "screen" output are:

```text
  Trace level (stdout):           "<date/time> Trace: <msg>"
  Debug level (stdout):           "<date/time> Debug: <msg>"
  Verbose level (stdout):         "<msg>""
  *Default*: Info|Print (stdout): "<msg>"
  Note level (stdout):            "Note: <msg>"
  Issue level (stdout):           "Issue: <msg>"
  Error level (stderr):           "Error: <msg>"
  Fatal level [stack] (stderr):   "Fatal: <msg>"

```

You can change anything about the default output values, prefixes, flags, etc and
even adjust where the output is sent.  You cannot adjust the hidden built-in
variable names used by API's and such (eg: LevelIssue), but all visible/output
can be adjusted.

For the default log file output we start with an ioutil.Discard for the
io.Writer which effectively means /dev/null to begin with.  However, if you
set a log file up using provided API's (as below) then the default log file
output will kick on and behave as follows:

```text
   Trace level:         "[<pid>] TRACE   <date/time> <shortfile:line#:shortfunc> Trace: <msg>"
   Debug level:         "[<pid>] DEBUG   <date/time> <shortfile:line#:shortfunc> Debug: <msg>"
   Verbose level:       "[<pid>] VERBOSE <date/time> <shortfile:line#:shortfunc> <msg>"
   Info|Print level:    "[<pid>] INFO    <date/time> <shortfile:line#:shortfunc> <msg>"
   Note level:          "[<pid>] NOTE    <date/time> <shortfile:line#:shortfunc> Note: <msg>"
   Issue level:         "[<pid>] ISSUE   <date/time> <shortfile:line#:shortfunc> Issue: <msg>"
   Error level:         "[<pid>] ERROR   <date/time> <shortfile:line#:shortfunc> Error: <msg>"
   Fatal level [stack]: "[<pid>] FATAL   <date/time> <shortfile:line#:shortfunc> Fatal: <msg>"

```

Again, all of this is adjustable so check out the next section.  To activate
a log file, as you'll see below, these are the two things to do:

1. Use an API call to prepare a temp or named file (will set up an io.Writer)
2. Use an API call to set the log level (so something is logged to your file)

Details below.

## Details on (optionally) configuring the 'out' package

To set up file logging or to adjust any of the defaults listed above
follow these examples:

### Send output to a temp log file, setting detailed output/logging thresholds

Enable all available levels of output to both the screen and to the temp log
file we just created (uses Go's temp file routines):

```go
    import (
        "github.com/dvln/out"
    )

    ...
    // set up a temp file for our tools output (we can print the
    // file name out at the end of our tools run so users know it),
    // call this early in your app if you want to log everything
    logFileName := out.UseTempLogFile("mytool.")

    ...
    // perhaps the CLI options map to these
    if Debug && Verbose {
        // lets set both screen and logfile to Trace level:
        out.SetThreshold(out.LevelTrace, out.ForBoth)
    }
    ...
```

After that all "out" package output functions (eg: out.Debugf or out.Println)
will go to the screens io.Writer and to the log file io.Writer using the default
settings (unless you have adjusted those yourself as below).

### Adjust the screen verbosity so we see Verbose level output

Quick note: by Verbose level output it is meant that the Verbose level and all
higher levels (higher numbers in the list above) will be shown, ie: Verbose,
Print, Note, Issue, Error, Fatal.  If one sets the level to Note then only Note,
Issue, Error and Fatal messages would be displayed to that target.

One should call any threshold setup (and log file setup and such) early in
your tool as output will only start flowing at that level after you have set
it up:

```go
    import (
        "github.com/dvln/out"
    )

    if Verbose {
        // set the threshold level for the screen to verbose output
        out.SetThreshold(out.LevelVerbose, out.ForScreen)
    }
    ...
```

Here we see that we used 'out.ForScreen' to indicate that this setting
is for the output stream going to the screen's io.Writer and has no effect
on any log file output stream settings/config (ie: if one is set up).

### Set log file output to a specific file to be at the Debug level

In this case we'll use another API to set up the logfile io.Writer to
a specific file:

```go
    import (
        "github.com/dvln/out"
    )
    ...
    // use the log file (currently Fatal's if problems setting up)
    out.SetLogFile("/some/dir/logfile")
    ...
    // you don't need this if of course, just pretending we had such a mode
    if Debug {
        // and sets set the log file threshold level to Debug
        out.SetThreshold(out.LevelDebug, out.ForLogfile)
    }
```

Aside: for Print/Info use "LevelInfo" as the name of the level.

### Examine a set of calls and how the output is formatted

This is a first foray into Go... I like spf13's jwalterweatherman output pkg
but I wanted a bit more independent control over the flags for each logger
and I found the 'log' packages handling of output formatting, multi-line
strings and strings with no newlines to not behave as I wanted (aside:
spf13's module uses Go's log package, this package started with that and
was changed over time after discovering what I felt were limitations).

Anyhow, an example with Note level calls (assumes our threshold is set to
print the Note level output):

```go
    import (
        "github.com/dvln/out"
    )
    ...
    out.Noteln("Successful test of: ")
    systemx := grabTestSystem()
    out.Notef("%s\n", systemx)
	out.Note("So I think you should\nuse this system\n")
```

This will come out cleanly with and without markup on the screen and in log
files with this package but using the stdlib Go 'log' it would insert newlines
for you after that 1st line and that just won't work (for me).  With this the
above will come out:

```text
Note: Successful test of: <somesystem>
Note: So I think you should
Note: use this system
```

If it was Debug instead of Note the screen output would be:

```text
<date/time> Debug: Successful test of: <somesystem>
<date/time> Debug: So I think you should
<date/time> Debug: use this system
```

If just a basic Print we would have:
```text
Successful test of: <somesystem>
So I think you should
use this system
```

The Go "log" package would insert newlines after each entry so 'somesystem' would
come up on the line below not giving the desired screen output and log file
mirroring of that output (assuming a log file was configured of course).  Also
log would put the prefix "Debug:" all the way to the left and I prefer it
essentially as part of the message (prepended by the package), assuming you
have a prefix set up for the given log level (you can drop prefixes or change
them if you prefer, see the SetPrefix() method).

### Adding in short filename and line# for screen debug level output:

```go
    import (
        "github.com/dvln/out"
    )
    ...
    // Note that out.Lstdflags is out.Ldate|out.Ltime (date/time), augment it:
	out.SetFlag(LevelDebug, out.Lstdflags|out.Lshortfile, ForScreen)
    ...
    // Set screen output so the 1st level of debugging is shown now:
    out.SetThreshold(out.LevelDebug, out.ForScreen)
    ...
    out.Debugln("Successful test of: ")
    systemx := grabTestSystem()
    out.Debugf("%s\n", systemx)
	out.Debug("So I think you should\nuse this system\n")
```

### Adding in long function names also for screen debug level output:

```go
    import (
        "github.com/dvln/out"
    )
    ...
    // Again, out.Lstdflags is the same as out.Ldate|out.Ltime (date/time)
    out.SetFlag(LevelDebug, out.Lstdflags|out.Lshortfile|out.Llongfunc, ForScreen)
    ...
    out.SetThreshold(out.LevelDebug, out.ForScreen)
    ...
    out.Debugln("Successful test of: ")
    systemx := grabTestSystem()
    out.Debugf("%s\n", systemx)
    out.Debug("So I think you should\nuse this system\n")
```

There is also Lshortfunc if you want just the function name and not the
package path included in the function name output.  Note that this is
the function name as returned by runtime.FuncForPC() for long form and
for short form we just grab the func name from the end of that.

### Replace the screen output io.Writer so it instead goes into a buffer

Switch the io.Writer for screen output to a buffer:

```go
    import (
        "github.com/dvln/out"
    )
	screenBuf := new(bytes.Buffer)
    // will set the screens io.Writer to the buffer
	out.SetWriter(screenBuf, out.ForScreen)
    ...
```

After that all output levels being sent to the screen will write into
the given buffer.

### Make the log file output exactly mirror the screen output, send to buffer

In this case we want to keep the screen output unchanged and going to the screen
and we want to instead turn on the log file output stream and make it match the
screens output exactly (by default logfile output has more flags turned on by
default to show pid, long date/timestamp, file name, line # and func name markup
and other goodies).  So we'll turn that off and match the screen output stream
flag settings and output threshold and write the result (matching the screen
output exactly) into a buffer:

```go
    import (
        "github.com/dvln/out"
    )

    // Lets match the default screen output stream flags setting.  We know
    // the output prefixes match by default so we won't change those, just
    // adjust the flag settings (note: out.Lstdflags = out.Ldate|out.Ltime):
    out.SetFlag(out.LevelAll, 0, out.ForLogfile) // clear all flag settings
    out.SetFlag(out.LevelTrace, out.Lstdflags, out.ForLogfile)
    out.SetFlag(out.LevelDebug, out.Lstdflags, out.ForLogfile)
    ...
    // Note: normally we wouldn't do the above to match the screen settings
    // as the default screen settings might have been changed by the client.
    // However, showing how to set flags directly as above is useful for
    // customizing an individual output stream and it's various levels.
    // To match flags we might instead do this:
    for lvl := out.LevelTrace; lvl <= out.LevelFatal; lvl++ {
        out.SetFlag(lvl, out.Flags(lvl, out.ForScreen), out.ForLogfile)
    }
    ...
    // Now match the screens output threshold (typically Print/Info level):
    out.SetThreshold(out.Threshold(out.ForScreen), out.ForLogfile)
    ...
    // Point the logfile output stream (it's io.Writer) at a buffer now:
    logfileBuf := new(bytes.Buffer)
    out.SetWriter(logfileBuf, out.ForLogfile)
    ...
    // Plenty of calls to 'out.Printf("%s\n", data)' or 'out.Issueln("Problem x!")'
    // would be done and you would want to eventually do something with the screen
    // output so grab it from the, configured above, "mirrored" log file buffer:
    screenOutputStr := logfileBuf.String().

```

### Use an io.Writer for screen/log file output (vs out.Print(), out.Debugf(), etc)

Use an 'out' package io.Writer for the debug and standard print levels so
that we can leverage any of the many packages/functions that need a writer
for output (available 'out' pkg writers are: TRACE, DEBUG, VERBOSE, INFO,
NOTE, ISSUE, ERROR, and FATAL, matching the log levels).  As to if the
screen or log file output streams actually prints them depends upon what
the output threshold settings are for each level of course.  Anyhow, one
can write directly to these streams as they are io.Writers:

```go
    import (
        "github.com/dvln/out"
    )
    fmt.Fprintf(out.DEBUG, "%s\n", someDebugString)
    fmt.Fprintf(out.INFO, "%s\n", someNormalToolOutputString)
    ...
```

The above is roughly the same as "out.Debugln(someDebugString)" followed
by an "out.Infoln(someNormalToolOutputString)" call.  However, if one is
using something like 'github.com/spf13/cobra' for a CLI commander one can
give it an io.Writer for any output so use of these 'out' package writers
can come in handy for this or the many other packages that take an io.Writer
(or one could write to a buffer io.Writer as shown above).

## Environment settings
There are some environment variables that can control the 'out' package
dynamically.  These are mostly useful for running a tool that uses this
package and more powerfully controlling debug output (limiting it to just
areas of interest), overriding default stack tracing behavior to perhaps
turn on stack traces for all issues as well as for dynamically adjusting
screen or logfile output flags to add more meta-data for troubleshooting:

 * PKG_OUT_DEBUG_SCOPE env can be set to to a list of packages or functions
   to restrict debug/trace output to, useful to trim debugging output and focus
   on a specific set of function(s) or packages(s).  This is basically just a
   substring match on the func name returned by runtime.FuncForPC() which tends
   to look like "github.com/jdough/coolpkg.Func" for example.  If your package
   is "github.com/jdough/mypkg" and you have kicked on debugging output in
   your tool and only want debugging from this pkg then set the env variable
   to "mypkg." and all other debug/trace output is not shown, if you want two
   packages then set it to "mypkg.,coolpkg." for example, if you want a pkg
   specific function then "mypkg.FuncA" could be used, etc.

 * PKG_OUT_LOGFILE_FLAGS and PKG_OUT_SCREEN_FLAGS env are used to dynamically
   tweak the screen or log "flags". This can be useful typically for adding in
   some flags for the screen output when debugging to see file/line#/function
   details inline withoutput (whatever flags you want, comma separated):
```text
   Predefined "group" settings, "debug" recommended really (LEVEL = output lvl):

     "debug"  : LEVEL time.microseconds shortfile:line#:shortfunc           : <output>
     "all"    : [pid] LEVEL date time.microseconds shortfile:line#:shortfunc: <output>
     "longall": [pid] LEVEL date time.microseconds longfile:line#:longfunc  : <output>

   Individual settings which can be combined (including to groups) are:

     "pid", "level", date", "time", "micro"|"microseconds", "file"|"shortfile",
     "longfile", "func"|"shortfunc", "longfunc" or "off".  Note that the
     "off" setting turns all flags off and trumps everything else if used.
```

 * PKG_OUT_STACK_TRACE_CONFIG can be set to "<targetstream>,<setting>" where
   the target steam can be "screen", "logfile" or "both" and the settings
   can be:

```text
     "nonzeroerrorexit": dump a stack trace on non-zero errors that cause exit
     "errorexit": one can dump an error and exit 0 (rarely), catch any error exit
     "allissues": any time an issue or error (exit or not) happens dump a stack trace
```

   One can also use the out.SetStackTraceConfig() API to set preferences within
   your tools.  The starting/default setting for this is:
```go
     out.SetStackTraceConfig(out.ForLogfile,out.StackTraceNonZeroErrorExit)
```
   So non-zero exits get dumped to your log file assuming one is configured
   to receive logging data at the right output thresholds and such.

# Current status
This is a very early release 0.2 level release.  It will be fluctuating but
should stabilize around Aug 2015.  It is recommended you vendor it as the API's
are not yet stable.  Definitely feel free to open issues under github as
needed (github.com/dvln/out).  Thanks for trying it "out"... yeah, couldn't
resist.  ;)

I've tried to use mutexes and atomic operations to protect the data so it can
be used concurrently but take that with a grain of salt as it's not heavily
tested in this area yet (passes race testing though).

I wrote this for use in [dvln](http://github.com/dvln). Yeah, doesn't really
exist yet but we shall see if I can change that.  It's targeted towards nested
multi-pkg multi-scm development line and workspace management (not necessarily
for Go workspaces but could be used for that as well)... what could go wrong? ;)
