out
===

Package for "leveled" CLI output printing to a terminal/screen with easy
mirroring of that output to a log file (all output via Go io.Writer's).
The goal is to be as easy to use as fmt.Printf, for example, but with
various levels, eg: out.Printf, out.Println, out.Print, out.Debugln,
out.Issueln, out.Issuef, out.Errorln, out.Errorf, out.Fatal, out.Fatalf, etc.
Combine this ease of use with independent control of prefixes, flags for
time/date, file/line# and func name, output thresholds and clean indenting and
somewhat reasonable formatting/prefixing for multi-line messages as well as for
non-newline terminated messages... and perhaps it's a package that might be of
use to some.

In a previous life I wanted a package/lib where I could push my output around
flexibily to screen, log file and redirecting it to capture buffers if desired
with flexible control over levels of output to each target and attached
meta-data to each target output stream (as well as each log level within that
output stream).  This package takes an early stab at that, to be refined with
use.

This started life as a wrapper around the Go standard 'log' library allowing two
Loggers to be independently controlled (one typically writing to the screen and
one to a log file), based on spf13's jwalterweatherman package (but looking to
give more independent control of screen vs log file meta-data and such)... but
upon further use I found 'log' to be too limiting in how it formats output and
jww to be a bit too restrictive in pushing to both output streams with different
threshold levels and flag-based add-on fields.  Hence, this package was created
to use io.Writers directly and tries to bring these benefits:

1. Ready for basic CLI screen output out of the box
2. Trivial to "turn on" logging (output mirroring) to a temp or named log file
3. Independent io.Writer control over screen and log file output thresholds
4. Independent control over flags (ie: add metadata like date/time, file/line#)
5. Clean alignment and handling for multi-line strings or strings w/no newlines
6. Avoids insertion of newlines into the screen or log file io.Writers
7. Ability to limit debug/trace to specific pkg(s) and/or function(s)
8. Ability to easily add stack traces on non-zero exit (eg: Fatal*) class errors
9. Attempts to be "safe" for concurrent use (this may need refining)

# Usage

## Basic usage:
Put calls throughout your code based on desired "levels" of messaging.
Simply run calls to the output functions:

 * out.Trace\[f|ln\](...)  
 * out.Debug\[f|ln\](...)
 * out.Verbose\[f|ln\](...)
 * out.Print\[f|ln\](...) or out.Info\[f|ln\](...)              (identical)
 * out.Note\[f|ln\](...)
 * out.Issue\[f|ln\](...) or out.IssueExit\[f|ln\](exitVal, ..) (2nd form exits)
 * out.Error\[f|ln\](...) or out.ErrorExit\[f|ln\](exitVal, ..) (2nd form exits)
 * out.Fatal\[f|ln\](...)                                       (always exits)

Each of these map to two io.Writers, one "defaulting" for the screen and the 
other targeted towards log file output (default is to discard log file output
until it is set up, see below).  One can, of course, redirect either or both
of these output streams anywhere via io.Writers (or multi-Writers).  An example
of standard usage:

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
    if err1 != nil {
        // Maybe this is a more severe unexpected error, but recoverable
        out.Errorf("File read failure (file: %s), ignoring ..\n", file)
    }
    ...
    if err2 != nil {
        // This is a fatal error, lets also add in a stack trace as maybe
        // this one should never happen and we'll need to troubleshoot:
		out.SetStacktraceOnExit(true)
        out.Fatalln(err2)
    }

```

There are 8 output levels but one can use just those that a given product needs
of course.  Personally I like debug and verbose options in my tools (typically).
If debug is used then Debug level for output is set, if debug and verbose both
are used then Trace level (more verbose debugging) is set up, etc.  I use a
"record" or "logfile" type option for logging if desired and one can use things
like a "quiet" or "silent" option to only see errors or important notes.  Use
whatever works for your tool and simply adjust the 'out' package levels to
match.  If you're needs are simpler check out Go's "log" and "fmt" packages
for basic output control or something like spf13's jwalterweatherman (jww)
package on github (github.com/spf13/jwalterweatherman), it's neat and some of
the work here is based on ideas from that (thanks to spf13) and the Go 'log'
package (thanks to the Go authors).

Aside: spf13 also created cobra for easy CLI setup and viper for config file
and environment variable settings/mgmt (ties nicely w/cobra).  I've modified
those to use this 'out' package instead of the original 'jww' package that
they were using, see 'github.com/dvln' for updated copies of those if you
want to leverage those tweaks.

The default settings add default "prefixes" on some of the messages, the
defaults for "screen" output are:

1. Trace level (stdout): "\<date/time\> Trace: \<message\>"
2. Debug level (stdout): "\<date/time\> Debug: \<message\>"
3. Verbose level (stdout): "\<message\>""
4. *Default*: Info|Print level (stdout): "\<message\>"
5. Note level (stdout): "Note: \<message\>"
6. Issue level (stdout): "Issue: \<message\>"
7. Error level (stderr): "ERROR: \<message\>"
8. Fatal level plus optional stacktrace (stderr): "FATAL: \<message\>"

The built-in names for these output levels can't be changed (unless you tweak
the code) but everything visible to the user can be such if you want the Issue
level to instead print "Error: " in mixed case (instead of "Issue: ") as the
prefix that can be done.  If you want no prefix or if you want time/date info
turned on/off that's also doable.  If you want everything to go to stdout or
if you want to change the default threshold to Verbose instead of Print/Info,
no problem.  Various adjustments via the API are covered below.

For the default log file output we start with an ioutil.Discard for the
io.Writer which effectively means /dev/null to begin with.  However, if you
set a log file up using provided API's (as below) then the default log file
output will kick on and behave as follows:

1. Trace level: "\<date/time\> \<file/line#\> Trace: \<message\>"
2. Debug level: "\<date/time\> \<file/line#\> Debug: \<message\>"
3. Verbose level: "\<date/time\> \<file/line#\> \<message\>"
4. Info|Print level: "\<date/time\> \<file/line#\> \<message\>"
5. Note level: "\<date/time\> \<file/line#\> Note: \<message\>"
6. Issue level: "\<date/time\> \<file/line#\> Issue: \<message\>"
7. Error level: "\<date/time\> \<file/line#\> ERROR: \<message\>"
8. Fatal level, optional stacktrace: "\<date/time file/line#\> FATAL: \<message\>"

Again, all of this is adjustable so check out the next section.  To activate
a log file, as you'll see below, these are the two things to do:

1. Use an API call to prepare a temp or named file (will set up an io.Writer)
2. Use an API call to set the log level (so something is logged to your file)

Lets see how to further configure and use this package next.

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
was changed over time after discovering limitations).

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

The log package would insert newlines after each entry so 'somesystem' would
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

## Environment settings
Currently there's only a couple:

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
   Predefined "group" settings, "debug" recommended really:

     "debug"  : time.microseconds shortfile:line#:shortfunc           : <output>
     "all"    : [pid] date time.microseconds shortfile:line#:shortfunc: <output>
     "longall": [pid] date time.microseconds longfile:line#:longfunc  : <output>

   Individual settings which can be combined (including to groups) are:

     "pid", "date", "time", "micro"|"microseconds", "file"|"shortfile",
     "longfile", "func"|"shortfunc", "longfunc" or "off".  Note that the
     "off" setting turns all flags off and trumps everything else if used.```

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
hasn't really been tested much.

I wrote this for use in [dvln](http://github.com/dvln). Yeah, doesn't really
exist yet but we shall see if I can change that.  It's targeted towards nested
multi-pkg multi-scm development line and workspace management (not necessarily
for Go workspaces but could be used for that as well)... what could go wrong? ;)
