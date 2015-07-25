out
===

A package for "leveled", easily "mirrored" (eg: logfile/buffer) and optionally
"augmented" (eg: pid, date/timestamp, Go file/func/line# data, etc) CLI output
stream management.  Designed to be a trivial drop-in replacement with no setup
required for all your output (eg: fmt.Println, fmt.Print and fmt.Printf would
become out.Println, out.Print and out.Printf with ability to also output at
various other levels like out.Verboseln, out.Debugf, out.Fatal, out.Issue,
out.Noteln, out.Printf, etc.  One can also access/adjust the screen and logfile
io.Writer's for any output level.  For debug and trace (ie: trace = verbose debug)
level output one can also control which package(s) or function(s) have their debug
info dumped (by default all debug data is dumped).  This can reduce debug output to
only those areas of code one wishes to focus on.

I've written CLI tools in the past where I wanted greater control of where I
send my output stream and how that stream might be dynamically "marked up",
split/mirrored/redirected and/or filtered via leveling.  To be able to easily
mirror screen output to a logfile or buffer (or bypass the screen entirely),
trivially, is powerful.  To be able to augment the screen or log file output,
independently, with additional meta-data (pid, date/timestamp, file/func info,
output level) and have all output remain cleanly aligned is very helpful for
troubleshooting.  Additionally, giving tool owners (or clients) the ability
to control what output "levels" are active and which pieces, if any, of add-on
meta-data are visible independently to each output stream adds value as does
the ability to dump stack traces on warnings and exits if/when desired and
to whichever output stream(s) desired.

Thanks much to spf13 (jwalterweatherman, etc), the Go authors (the log package)
as well as Dropbox folks and their open error package.  Ideas from these and
others have been munged together to create this package.

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
11. Support for plugin formatters to roll your own format (can even support JSON output modes)
12. Optional: "detailed" errors type adds stack from orig error instance, wrapping of errors

Note: more documents on the last couple of options will be forthcoming.

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
  Trace level (stdout):           "\<date/time\> Trace: \<msg\>"
  Debug level (stdout):           "\<date/time\> Debug: \<msg\>"
  Verbose level (stdout):         "\<msg\>""
  *Default*: Info|Print (stdout): "\<msg\>"
  Note level (stdout):            "Note: \<msg\>"
  Issue level (stdout):           "Issue: \<msg\>"
  Error level (stderr):           "Error: \<msg\>"
  Fatal level \[stack\] (stderr):   "Fatal: \<msg\>"

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
   Trace level:         "\[\<pid\>\] TRACE   \<date/time\> \<shortfile:line#:shortfunc\> Trace: \<msg\>"
   Debug level:         "\[\<pid\>\] DEBUG   \<date/time\> \<shortfile:line#:shortfunc\> Debug: \<msg\>"
   Verbose level:       "\[\<pid\>\] VERBOSE \<date/time\> \<shortfile:line#:shortfunc\> \<msg\>"
   Info|Print level:    "\[\<pid\>\] INFO    \<date/time\> \<shortfile:line#:shortfunc\> \<msg\>"
   Note level:          "\[\<pid\>\] NOTE    \<date/time\> \<shortfile:line#:shortfunc\> Note: \<msg\>"
   Issue level:         "\[\<pid\>\] ISSUE   \<date/time\> \<shortfile:line#:shortfunc\> Issue: \<msg\>"
   Error level:         "\[\<pid\>\] ERROR   \<date/time\> \<shortfile:line#:shortfunc\> Error: \<msg\>"
   Fatal level \[stack\]: "\[\<pid\>\] FATAL   \<date/time\> \<shortfile:line#:shortfunc\> Fatal: \<msg\>"

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
	out.Note("So I think you should\nbuy this system as it\nis a quality system\n")
```

This will come out cleanly with and without markup on the screen and in log
files with this package but using the builtin 'log' it would insert newlines
for you after that 1st line and that just won't work (for me).  With this the
above will come out:

```text
Note: Successful test of: <somesystem>
Note: So I think you should
Note: buy this system as it
Note: is a quality system
```

If it was Debug instead of Note the output would be:

```text
<date/time> Debug: Successful test of: <somesystem>
<date/time> Debug: So I think you should
<date/time> Debug: buy this system as it
<date/time> Debug: is a quality system
```

If just a basic Print we would have:
```text
Successful test of: <somesystem>
So I think you should
buy this system as it
is a quality system
```

The Go log package would insert newlines after each entry so 'somesystem' would
come up on the line below not giving the desired screen output and log file
mirroring of that output (assuming a log file was active at this level).  Also
log would put the prefix "Debug:" all the way to the left and I prefer it
essentially as part of the message (prepended by the package), assuming you
have a prefix set up for the given log level (you can override these, see
the SetPrefix() method).

### Adding in short filename and line# for screen debug level output:

```go
    import (
        "github.com/dvln/out"
    )
    ...
    // Note that out.Lstdflags is out.Ldate|out.Ltime (date/time)
	out.SetFlag(LevelDebug, out.Lstdflags|out.Lshortfile, ForScreen)
    ...
    out.SetThreshold(out.LevelDebug, out.ForScreen)
    ...
    out.Debugln("Successful test of: ")
    systemx := grabTestSystem()
    out.Debugf("%s\n", systemx)
	out.Debug("So I think you should\nbuy this system as it\nis a quality system\n")
```

### Adding in long function names also for screen debug level output:

```go
    import (
        "github.com/dvln/out"
    )
    ...
    // Note that out.Lstdflags is out.Ldate|out.Ltime (date/time)
    out.SetFlag(LevelDebug, out.Lstdflags|out.Lshortfile|out.Llongfunc, ForScreen)
    ...
    out.SetThreshold(out.LevelDebug, out.ForScreen)
    ...
    out.Debugln("Successful test of: ")
    systemx := grabTestSystem()
    out.Debugf("%s\n", systemx)
    out.Debug("So I think you should\nbuy this system as it\nis a quality system\n")
```

There is also Lshortfunc if you want just the function name and not the
package path included in the function name output.  Note that this is
the function name as returned by runtime.FuncForPC() for long form and
for short form we just grab the func name from the end of that.

### Replace the screen output io.Writer so it instead goes into a buffer

Lets switch the io.Writer used for the screen so it goes into a byte buffer:

```go
    import (
        "github.com/dvln/out"
    )
	screenBuf := new(bytes.Buffer)
    // will set screen io.Writer to the buffer
	out.SetWriter(screenBuf, out.ForScreen)
    ...
```

### Replace the log file outputs io.Writer so that instead goes into a buffer

Another option: leave the screen alone and use the log file writer as a buffer:

```go
    import (
        "github.com/dvln/out"
    )

    // First lets turn off the log file defaults so we match the screen
    // output defaults (ie: only timestamps for Trace/Debug, otherwise
	// no timestamps or filenames/etc added in), note that the LevelAll
    // impacts all levels whereas specific levels only impact that specific
    // given level (could be a bit flag later).
	out.SetFlag(LevelAll, 0, out.ForLogfile)
	out.SetFlag(LevelTrace, out.Lstdflags, out.ForLogfile)
	out.SetFlag(LevelDebug, out.Lstdflags, out.ForLogfile)
    ...
    // Now lets set up a buffer for the "logfile" output (it is, of course,
    // no longer a log file really, it's a buffer... but to this package the
    // io.Writer I'm mucking with is the "log file" writer:
	logfileBuf := new(bytes.Buffer)
	out.SetWriter(logfileBuf, out.ForLogfile)
    ...
```

### Use an io.Writer for screen/logfile output (vs out.Print, out.Debug, etc)

Use an 'out' package io.Writer for the debug and standard print levels so
that we can leverage any of the many packages/functions that need a writer
for output (available: TRACE, DEBUG, VERBOSE, INFO, NOTE, ISSUE, ERROR,
and FATAL are all available io.Writer's in the 'out' package):

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
could come in handy for this or many other packages that take an io.Writer
(or one could write to a buffer io.Writer and in the calling tool decide
if it's normal or error output and then write at the correct output level
which is what I would actually recommend so this example isn't the best
frankly but there are many io.Writer uses regardless).


## Environment settings
Currently there's only a few:

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
```go
   Predefined "group" settings, "debug" recommended really (LEVEL = output lvl):

     "debug"  : LEVEL time.microseconds shortfile:line#:shortfunc           : <output>
     "all"    : [pid] LEVEL date time.microseconds shortfile:line#:shortfunc: <output>
     "longall": [pid] LEVEL date time.microseconds longfile:line#:longfunc  : <output>

   Individual settings which can be combined (including to groups) are:

     "pid", "level", date", "time", "micro"|"microseconds", "file"|"shortfile",
     "longfile", "func"|"shortfunc", "longfunc" or "off".  Note that the
     "off" setting turns all flags off and trumps everything else if used.
```

 * PKG_OUT_NONZERO_EXIT_STACKTRACE env set to "1" causes stacktraces to kick in
   for any non-zero exit done through this package (os.Exit() is not affected),
   so basically if you use IssueExit... or ErrorExit... or Fatal... it works

And one meant for internal use (eg: for testing purposes):

 * PKG_OUT_NO_EXIT env set to "1" causes bypass of os.Exit() in this package,
   only applies to exits reached via the 'out' package

# Current status
This is a very early release... needs more work so use with caution (vendor it)
as it's likely to change a fair bit over the coming months.  Thanks again to
spf13 and the Go authors for some ideas here and definitely feel free to send
in any issues via Github and I'll try and knock em out.

I've tried to use mutexes to protect the data similar to Go 'log' but that
hasn't, frankly, been tested much.

I wrote this for use in [dvln](http://github.com/dvln). Yeah, doesn't really
exist yet but we shall see if I can change that.  It's targeted towards nested
multi-pkg multi-scm development line and workspace management (not necessarily
for Go workspaces but could be used for that as well)... what could go wrong? ;)
