out
===

A package for powerful "leveled" output designed to do more than Go's standard
log package.  This package has two io.Writer streams that are independently
controllable.  These are typically for screen output (which works out of
the box) and mirrored log file output which is trivial to set up for a tmp
or named log file.  With independent control over the output streams one
can print regular output to the screen while mirroring that to a log file
or one can easily set the log file output to be at a different output
level (eg: 'trace' level for verbose debugging) with markup data added
such as Go file name, func name and line #, date/timestamp, pid, log
level, etc prefixed.  All flags/metadata are fully configurable (similar
to Go's log package, just with more configuration/markup options).

Basically replace calls to 'fmt.Println()' or 'log.Printf()' instead with
calls to 'out.Println()' and 'out.Printf()'.  If one wanted to use different
levels one could use 'out.Debugln()' or 'out.Notef()' or various other
output levels just as easily with the 'out.\<Level\>\[ln|f\]()' routines. 
One can do additional things as well, a few examples:

1. Errors could be set up to be shown in human format or JSON (see formatters)
2. Trim/filter debugging output to specific packages or even package functions
3. Optional "detailed" errors to get stack traces near orig error occurance
4. Cleaner log file vs screen output formatting/alignment (vs Go 'log' pkg)
5. Augment screen and/or logged data with more extensive meta-data markup options
6. Direct access to io.Writers for every output level (eg: out.TRACE, out.DEBUG) 
7. All of these and more are optional but add power if needed, more below ...

A CLI tool that uses 'out' can easly have screen output that looked something
like the following while the log file output showed more detail:

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

Screen output is clean, in the log file one sees details about each line of
screen output such as the pid, the general output level (print=info), the
date/time, the go file and line # for each output call as well as the function
name... along with the output formatted to align fairly well so it's easy to
compare the log output to the screen output (ie: see what the user sees).  If
long filenames were desired with pkg name info that is available as well, same
with longer function name information.  If desired trace/debug could be easily
added to the log file output stream while not showing them on the users screen
stream.  Independent control can be powerful.

Optionally available is something called "detailed" errors.  If one wants stack
traces closer to an original error occurrance these can be useful (similar to
how Dropbox does errors, borrowed from their ideas/code, thanks!).  Additionally,
if one wants error codes one can optionally use them.  One can also continue
to "check equality" with core library or vendor pkg error message "contants" even 
if one has wrapped the original error as it is returned through the call stack.
Keep in mind that use of detailed errors with optional error codes is not required
in any way since one can leverage the "out" package simply for levelled output alone
without using these.

The key "out" package goal is flexible control of output streams.  Flexible
mark-up, dynamically redirection, meta-data augmentation to help with support
or troubleshooting, etc.  If something fancier is needed one can dynamically
plug in one's own output formatter with this package to override or augment
the built-in output prefixes and flag based meta-data.

A more complete list of features:

1. Ready for basic CLI screen output with levels out of the box, no setup
2. Drop-in replacement for log or fmt's Print[f|ln](), w/multiple level support
3. Trivial to "turn on" logging (output mirroring) to a temp or named log file
4. Independent io.Writer control over the two output streams (eg: screen/logfile)
5. Independent output level thresholds for each target (eg: screen/logfile)
6. Independent control over flags (eg: augment log file with date/time, file/line#, etc)
7. Clean alignment and handling for multi-line strings or strings w/no newlines
8. Ability to limit debug/trace output to specific pkg(s) or function(s)
9. Ability to easily add stack trace on issues/errors/fatal's (dying/non-zero or not)
10. Tries to be "safe" for concurrent use (if any problems, please open a git issue)
11. Support for plugin formatters to roll your own format (or support other output mechanisms)
12. "Deferred" function can be set to run before exit (eg: summary info, tmp logfile name, etc)
13. Optional "detailed" errors type gives more accurate stack traces, optional error codes

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
desired level.  You can give the pkg any io.Writer for any output level for
both the screen and log file output, so you have control.  Additionally, if
you just want an io.Writer for an existing output level you can use the
built-in writers as another way to send output via the 'out' package, see
the out.TRACE, out.DEBUG, out.VERBOSE, out.INFO, out.NOTE, out.ISSUE,
out.ERROR and out.FATAL io.Writers that are directly accessible, eg:

 * fmt.Fprintf(out.DEBUG, "%s", someDebugString)    (same as out.Debug(..))
 * fmt.Fprintf(out.INFO, "%s", someString)          (same as out.Print|out.Info)
 * ...

One can also access the writers via API but not as easy as the above:

 * fmt.Fprintf(out.GetWriter(out.LevelInfo), "%s", someString)
 * fmt.Fprintf(out.GetWriter(out.LevelDebug), "%s", someDebugString)
 * ...

See out.SetWriter() examples below on changing screen or log file
writers as well as SetThreshold() to change output thresholds for
each writer and such.

An example of standard usage:

```go
    import (
        "github.com/dvln/out"
    )

    // Lets set up a tmp log file first
    logFileName := out.UseTempLogFile("mytool.")
    // Print that to the screen as log file currently discarding output
    out.Println("Temp log file:", logFileName)

    // Set log file output threshold: verbose, info/print, note, issue, err, fatal:
    out.SetThreshold(out.LevelVerbose, out.ForLogfile)
    // Note that stack traces for non-zero exit (IssueExit/ErrorExit/Fatal)
    // are set up,by default, to go to the log file (which we just config'd)

    ...
    // Day to day normal output
    out.Printf("some info: %s\n", infostr)

    // Less important output not shown to screen (LevelInfo default) but it
    // will go into the log file (LevelVerbose as above)
    out.Verbosef("data was pulled from src: %s\n", source)

    ...
    // Will be dumped as 'Note: Keep in mind ...', coming one output
    // level above the Print/Info routines (LevelInfo) as LevelNote:
    out.Noteln("Keep in mind that when this is done you need to do z")

    ...
    if err != nil {
        // Something like an issue that, if it sticks around, one might want
        // to contact the infra team to fix before it keeps is from working:
        //   Issue: Some expected compute resources offline due to unexpected issue:
        //   Issue:   <somerr>
        //   Issue: If this continues for a while let IT know, continuing...
        // If we wanted a stack trace for all issues to the log file we
        // could have set this here or above and we would get a stack
        // in the log file and the tool would keep running:
        //   out.SetStackTraceConfig(out.ForLogfile|out.StackTraceAllIssues)
        // If you want it on the screen also use out.ForBoth instead.
        out.Issuef("Some expected compute resources offline due to unexpected issue:\n  %s\n", err)
        out.Issueln("If this continues for a while let IT know, continuing...")
    }
    ...
    if err != nil {
        // Maybe this is a more severe unexpected error, but recoverable
        out.Errorf("Unexpected File read failure (file: %s), WTF, bypassing.\n  Error: %s\n", file, err)
    }
    ...
    if err != nil {
        // With above settings a stack trace will go to the log file along
        // with this error message above it.  Note that one can dynamically
        // add/rm stack traces at runtime as well via PKG_OUT_STACK_TRACE_CONFIG
        out.Fatalln(err)

        // Aside: maybe you don't like the 'Fatal: <msg>' prefix that will add,
        // you can remove that via SetPrefix() or you can exit out of an Issue
        // or error level message if you prefer those prefixes (but it will be
        // at a less severe level), see IssueExit[f|ln]() and ErrorExit[f|ln]()
    }

```

Pretty straightforward.  Your code can get/set output thresholds (which levels
are printed to the screen or to a log file, independently), config stack traces
and when to see them, remove/add/change message prefixes, set flags to mark up
screen or log file output with timestamps/filename/line#'sa, re-route where
data is written to (via io.Writer's), plug in your own formatter if you don't
like these built-in options (which can reformat, suppress/re-route output, etc)
and even use built-in "detailed" errors if preferred (more below).  Note also
that one can easily hook up CLI options like a debug or verbose option into
API's to set output thresholds and turn on get log file names and such.

There are 8 output levels (perhaps too many for most folks) but one can, of
course, just use those that a given product needs.  There is no need to use
levels you do not want.

Quick note: some packages like spf13's 'viper' have been ported to 'out' in
the 'dvln' organization, see 'http://github.com/dvln' for those.

The default configuration of 'out' sets up default "prefixes" on some of
the output levels.  The defaults for "screen" output out of the box are:

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

You can change anything about the default output values, prefixes, flags, etc
and adjust where the output is sent.  You cannot adjust the hidden built-in
level names used by API's (unless you tweak the code, eg: LevelIssue), but
all client visible output can be adjusted (so if you prefer "Warning: " as
a prefix for the Issue level that is easy to tweak.

For the default log file output io.Writer this starts with an ioutil.Discard
which effectively means send output to /dev/null even if the logging threshold
says to log.  The default logging threshold is also set to discard data send to
the log file output stream.  However, once these two items are set up (as in
examples above and below) then this is the default config for the various
levels for a log file:

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

Once that is done use an API to set the desired output level for the log file
and you are set.  More details below.

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
        // lets set both screen and log file to Trace level:
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

In this case we'll use another API to set up the log file io.Writer to
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
screens output exactly (by default log file output has more flags turned on by
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
    // Point the log file output stream (it's io.Writer) at a buffer now:
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

### Set up a Formatter and adjust or redirect the info to be dumped

Formatters can be attached at any output level (or to all output levels).
They are currently independent of output target meaning they get the raw
message that is being given (to screen or out), even before it's determined
if screen or log file output is active (based on thresholds and such) and
before the message has been prefixed or augented with flag metadata.  The
formater can change the message, augment it, make it empty and can also
tell the 'out' package to skip prefixes and flags based meta-data additions
if desired... and can even tell the 'out' package NOT to dump the message
to either screen or log file or both if desired.  A formatter is a Go
interface so if you implement the interface with an empty struct for
example then one can instantiate that and use SetFormatter() to attach
it to any of the log levels (or all of them).  Note that the formatter
will be told if this is a terminal type issue (IssueExit(), ErrorExit()
or Fatal()) so you can behave correctly if dying or not.

One would use a custom formatter if one wished to do something like:
1. Skip all output prefixing and meta-data markup and use my own custom setup
2. Dynamically morph a message (eg: add error code to msg, morph it to JSON)
3. Take the message and NOT dump it to screen and/or log, redirect it elsewhere
4. Whatever else you can think of 

I created this because my CLI has text and JSON output modes and I wanted
my Issue/Error/Fatal calls to be dumped in a JSON compatible way (so my errors,
no matter where they are done, end up coming out as JSON when I'm in that
mode).  To do this I attached a formatter to the Issue/Error/Fatal level
and I handle dying issues/errors/fatals in that formatter differently than
non-terminal issues/errors.  Dying errors form up a small/tight JSON output
that includes the 'error', and the msg, error level (ISSUE/ERROR/FATAL) 
and error code (if one is available, defaults to 100, see next section
on detailed errors and their optional use with this package).  Any non
fatal type of Issue or Error gets stored as a 'warning' in my final JSON
structure and is not printed at all at this time to the screen or log file
as the "final" JSON will be dumped at the end with the 'warning' field in
it (filled in with the msg, level and code if one is available).

Anyhow, I tried to make it generic so you can just reformat messages or
override build-in formatting when desired... probably the more common use
case.  Here's how one might add this in:

```go
    import (
        "github.com/dvln/out"
    )
    ...
    type mungeOutput struct{}

    // FormatMessage takes the following params:
    // - msg (string): The raw message coming from the clients call (eg: "out.Println(msg)")
    // - outLevel (Level): This is the output level (out.LevelTrace, out.LevelDebug, etc), one
    //   can get a text representation via 'fmt.Sprintf("%s", outLevel)' if needed (eg: "TRACE")
    // - code (int): An error code, defaults to the 'out' pkg default if err codes not in use (100)
    // - stack (string): If the Issue/Error/Fatal level then the "best" stack trace available is
    //     passed in (prefixed with "\nStack Trace: <stack>").  Normally don't use this since the
    //     built-in stuff will "smart" add it based on if stack traces are active or not to the
    //     given output target.  However, here it is if you need it.
    // - dying (bool): Will be true if this is a terminal situation, IssueExit(), ErrorExit() or
    //     Fatal() has been called... as this may effect what you do or return (see example above)
    func (f mungeOutput) FormatMessage(msg string, outLevel Level, code int, stack string, dying bool) (string, int, bool) {
        // Lets add "[fun fun fun]" to the end of all our messages:
        msg = fmt.Sprintf("%s [fun fun fun]", msg)

        // One can suppress out.ForBoth, out.ForLogfile, out.ForScreen but I
        // won't suppress any output, I'm just munging the message a little:
        suppressOutputMask := 0

        // I will not suppress any native 'out' pkg formatting so I'll still
        // get my 'Error: ' prefix (if this was an error) and I'll still get
        // any flags meta-data added to my msg, same as would have happened
        // if a formatter wasn't used and a msg came through
        suppressNativePrefixing := false

        // That's it... return the msg to use and if anything should be suppressed
        return msg, suppressOutputMask, suppressNativePrefixing
    }

    func main () {
        ...
        var myOutputTweaker mungeOutput
        out.SetFormatter(out.LevelAll, myOutputTweaker)

        ...
        out.Print("This is fun")
        out.Error("Something went wrong!")
        out.Exit(0)
    }
```

That would implement the Formatter interface and all your output calls
would get routed through your formatter, adjusting them as desired (and
possibly suppressing built-in formatting and prefixing and such or even
preventing output if desired from the 'out' package).

### Setting up a "deferred" function to call before terminating

One can register a single function to be called just before your tool
will exit.  This only works if one is always exiting via calls to 'out.Exit()'
or out.Fatal() or the out.*Exit() functions.  Use is pretty easy:

```go
// mySummary wraps up my programs output with summary data or perhaps
// by closing or moving files or rotating log files or outputting names
// of tmp files generated within the tool to capture output.
func mySummary(exitVal int) {
    // For our example we'll just note a tmp log file path/name but we'll
    // make sure that msg only goes to the screen output on STDERR and is
    // not written to the log file output stream... only resets the output
    // stream if we're writing to stdout, if we're writing to some other
    // output io.Writer for the note level output we'll just use that:
    if tmpLogfileMsg != "" {
        // Send screen note to STDERR if currently it is the default (STDOUT)
        currWriter := out.Writer(out.LevelNote, out.ForScreen)
        if currWriter == os.Stdout {
            out.SetWriter(out.LevelNote, os.Stderr, out.ForScreen)

        }

        // Don't put the tmp log file note in the log file itself...
        currThresh := out.Threshold(out.ForLogfile)
        out.SetThreshold(out.LevelDiscard, out.ForLogfile)

        out.Noteln(tmpLogfileMsg)

        // To be safe set them back to previous settings (even though exiting)
        out.SetThreshold(currThresh, out.ForLogfile)
        out.SetWriter(out.LevelNote, currWriter, out.ForScreen)
    }
}

func main () {
    ...
    out.SetDeferFunc(mySummary)
}
```

If you want to clear it set it to nil via SetDeferFunc() and if you want to
see if it's currently set and what it is set to use DeferFunc() to get it.
You should set this up as soon as you have a need for some pre-exit function.
Again, it will NOT fire for os.Exit() called directly (or indirectly by 
something like log.Fatal() and such, only works with the 'out' pkg exit
mechanisms).

### Using detailed errors for your errorring (optional, not required!!!)

To create a new detailed error one would use one of the following:

```text
  Without using error codes:              With using error codes:
  -------------------------------------   ----------------------------------------
  out.NewErr("Some error message")        out.NewErr("Some error message", code)
  out.NewErrf(0, "Some error: %s", msg)   out.NewErrf(code, "Some error: %s", msg)
```

Yeah, not great for NewErrf, but you get the idea... variadic args and all.
Anyhow, these will store the error message, the error code and a stack trace
from where this was called directly in the detailed error.

Another way to create a detailed error is to wrap a regular Go error with
a detailed error... one can also wrap another detailed error as an error
is passed back through routines.  Usage for wrapping:

```text
  Without using error codes:                   With using error codes:
  -------------------------------------------  ----------------------------------------------
  out.WrapErr(err, "Some error message")       out.WrapErr(err, "Some error message", code)
  out.WrapErrf(err, 0, "Some error: %s", msg)  out.WrapErrf(err, code, "Some error: %s", msg)
```

Like a new error a wrapped error stores the original error as
the "inner" error, stores the new detailed error msg, error code
and a stack trace from where this was called.  You can continue
to wrap errors again and again if it's helpful for your needs.
When you dump an error it will traverse all wrapped errors down
to the initial or root error and show the newest to the oldest
errors (unless you ask for a "shallow" message then only the
most recent message will be shown).  As for stack traces it
will use the oldest error that has a stack trace included as
that will be closest to when the problem first occurred and give
you the easiest troubleshooting.

If you want to see if a detailed error "contains" an error that is
set up in the standard library (for example) you can use this:
```go
    if IsError(myErr, ErrConstName) {
        // if the detailed errors "root" error matches ErrConstName IsError()
        // will be true and you'll match
    }

    // If you are using error codes you can also match on error codes:
    if IsError(myErr, nil, 500) {
        // if error code 500 it used in *any* of the nested errors
        // in my detailed error then IsError() will return true
    }
```

For my tools I plan on using detailed errors for all my errors and I
will wrap "core" stdlib class errors as quickly as possible within the
routine that experienced them before passing them back so I have a stack
trace that points directly to where the issue started from (even if it 
was passed down/back through 3 or 4 routines before being printed the stack
will still be clear as to the original location of the issue).

Here is a fuller example of what will happen if you use the detailed error
feature and you pass those errors into Issue, Error or Fatal related routines
in the 'out' package:

```go
    ...
    func tryInnerFunc() error {
        ...
        if err := someStdlibFileOpenCall(...); err != nil {
            return out.WrapErr(err, "Problem: related to opening mytool config file:", 2040)
        }
    }

    func tryMiddleFunc() error {
        ...
        if err := tryInnerFunc(); err != nil {
            return out.WrapErr(err, "Failure occurred during \"middle functionality\".", 3010)
        }
    }

    func main() {
        ...
        // Maybe we think this is too generic and so we won't give it an error number
        // but we'll still drop in an overall top level class message
        out.SetStackTraceConfig(out.ForBoth|out.StackTraceNonZeroErrorExit)
        if err := tryMiddleFunc(); err != nil {
            out.Fatal(out.WrapErr(err, "Tool unable to complete requested task.")

            // The above could just as easily have been 'out.Fatal(err)' if one
            // did not want that high level message at all.
        }
    }
```

It would use the "highest" error code (3010) for the message although IsError()
would match on 2040 and 3010 both as well as the original system error.  The
output of the Fatal call would be something like:

```text
Fatal #3010: Tool unable to complete requested task.
Fatal #3010: Failure occurred during "middle functionality".
Fatal #3010: Problem: related to opening mytool config file:
Fatal #3010: <system error returned from file open call>
Fatal #3010: 
Fatal #3010: Stack Trace: go routing 1 [running]:
Fatal #3010: ...[stack trace pointing at tryInnerFunc() WrapErr call location]...
```

With that our stack trace points to where the problem really occurred and
the rest of our output gives a clean indication of what happened all the
way down to the actual stdlib error that occurred and couches that within
other error wraps (optional of course except for that WrapErr() right
where the original error occurred, that's where that really needs to be).

Note: if you change the prefix via SetPrefix() so "Issue: ", "Error: "
or "Fatal: " drops the : or has more than one : then the error code
auto-insertion will not happen.  Also note that the default error code
which starts out as 100 but you can change that, will not be printed if
that is set (nor wlll an error code of 0, consider both "reserved"
although you can change the default code of 100, see SetDefaultErrCode()
if needed).

## Environment settings
There are some environment variables that can control the 'out' package
dynamically.  These are mostly useful for running a tool that uses this
package and more powerfully controlling debug output (limiting it to just
areas of interest), overriding default stack tracing behavior to perhaps
turn on stack traces for all issues as well as for dynamically adjusting
screen or log file output flags to add more meta-data for troubleshooting:

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
     out.SetStackTraceConfig(out.ForLogfile|out.StackTraceNonZeroErrorExit)
```
   So non-zero exits get dumped to your log file assuming one is configured
   to receive logging data at the right output thresholds and such.

# Current status
This is now stabilizing.  Currently 'out' is at a "v0.8.0" level (semantic
versioning v2).  Not fully stable until v1.0.0 so keep that in mind and
fork it if you're using it (or vendor it).  Expect v1.0.0 around early
2016 after I get some more use and input.  Feel free to fork, send pull
requests or file issues.

Note: I've tried to use mutexes and atomic operations to protect the data
so it can be used concurrently but take that with a grain of salt as it's
not heavily tested in this area yet (yes, it passes race testing).

I wrote this for use in [dvln](http://github.com/dvln/dvln). Yeah, doesn't really
exist yet but we shall see if I can change that (but one can see this package
in use in that and packages it depends upon).  It's targeted towards nested
multi-pkg multi-scm development line and workspace management (not necessarily
for Go workspaces but could be used for that as well)... what could go wrong? ;)

Thanks again to the Go authors and various others like spf13, Dropbox and countless
others authors with open code that I have gleaned ideas and code from to generate this
package.

