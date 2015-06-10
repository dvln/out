out
===

A package for "leveled", easily "mirrored" (eg: logfile/buffer) and optionally
"augmented" (eg: pid, date/timestamp, Go file/func/line# data, etc) CLI output
stream management.  Designed to be a trivial drop-in replacement with no setup
required for all your output (eg: fmt.Println, fmt.Print and fmt.Printf would
become out.Println, out.Print and out.Printf with ability to also output at
various other levels like out.Verboseln or out.Debugf or out.Fatal or out.Issue
and various other levels).  One can also access an io.Writer for any output
level (if needed).  For debug and trace (ie: trace = verbose debug) level
output one can also control which package(s) or files have their debug info
dumped (assuming debug output is active one can filter that by pkg/file also).

I've written CLI tools in the past where I wanted greater control of where I
send my output stream and how that stream might be dynamically "marked up",
split/mirrored/redirected and/or filtered via leveling.  To be able to easily
mirror screen output to a logfile or a buffer (or skip the screen entirely),
trivially, is powerful.  To be able to augment the screen or log file output,
independently, with additional meta-data (pid, date/timestamp, file/func info,
output level) and have all output remain cleanly aligned is very helpful for
troubleshooting.  Additionally, giving tool owners (or clients) the ability
to control what output "levels" are active and which pieces, if any, of add-on
meta-data are visible, independently, to each output stream can be valuable.

Example: user screen output set at a "normal level" while, at the same time,
that same tool "run" is being transparently logged to a file with verbose
debugging levels of output also being added there along with pid's/timestamps
for each line of output identified, the output "log level" for each line of
output, and even Go source filename/func/line# added to see where those output
calls are coming from.  All of this can really add up in helping with
troubleshooting as well as performance analysis and even visibility of user
"setup" details.  For example, if your verbose debug output gives CLI info,
env settings, cfg file settings, etc then one could reproduce the users cmds
and setup and even timing if desired using that logfile data.  Another example:
for testing I want to push screen output and logfile output into buffers I can
easily check and test against, no "real" screen output needed but I want to
verify that if the tool was really running exactly what was going to each
target stream (see out_test.go).

This started life as a wrapper around the Go standard 'log' library allowing two
Loggers to be independently controlled (one typically writing to the screen and
one to a log file), based on spf13's jwalterweatherman package (but looking to
give more independent control of screen vs log file meta-data and such)... but
upon further use I found 'log' to be too limiting in how it formats output and
jww to be a bit too restrictive via it's io.Multiwriter use (same with log) in
that I was not able to independently control the output stream levels and markup
meta-data and such.  Hence, this package was created with these goals in mind:

1. Ready for basic CLI screen output with levels out of the box, no setup
2. Easy drop-in replacement for fmt.Print[f|ln](), plus level specific func's
3. Trivial to "turn on" logging (output mirroring) to a temp or named log file
4. Independent io.Writer control over the two output streams (eg: screen/logfile)
5. Independent output level thresholds for each target (eg: screen and logfile)
6. Independent control over flags (ie: add metadata like date/time, file/line#)
7. Access to io.Writer for any output level (streams/markup based on curr setup)
8. Clean alignment and handling for multi-line strings or strings w/no newlines
9. "Smarter" insertion of newlines into the screen or log file io.Writers
10. Ability to limit debug/trace output to specific pkg(s) and/or function(s)
11. Ability to easily add stack trace on non-zero exit (eg: Fatal*) class errors
12. Attempts to be "safe" for concurrent use (currently lacks thorough testing)

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
other usually targeted towards log file output (default is to discard log file
output until it is configured, see below).  One can, of course, redirect either
or both of these output streams anywhere via io.Writers (or multi-Writers).
One can also send output to the two output streams via an io.Writer at the
desired level, there are a couple ways to get a writer but the easiest way
is to use out.TRACE, out.DEBUG, out.VERBOSE, out.INFO, out.NOTE, out.ISSUE,
out.ERROR, out.FATAL directly:

 * fmt.Fprintf(out.DEBUG, "%s", someDebugString)    (same as out.Debug(..))
 * fmt.Fprintf(out.INFO, "%s", someString)          (same as out.Print|out.Info)
 * ...

One can use an API to get a writer based on a Level type if desired:

 * fmt.Fprintf(out.GetWriter(out.LevelInfo), "%s", someString)
 * fmt.Fprintf(out.GetWriter(out.LevelDebug), "%s", someDebugString)
 * ...

Just like the function calls the io.Writer for any level goes to the same two
underlying io.Writers.  Also like the function all interface the output to those
two writers and any attached meta-data functions the same (ie: you'll get output
if the selected output level is high enough based on your thresholds, etc).

An example of standard usage via the functions:

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
        // This is a fatal error, lets also add in a stack trace as maybe
        // this one should never happen and we'll need to troubleshoot:
		out.SetStacktraceOnExit(true)
        out.Fatalln(err)
    }

```

There are 8 output levels (perhaps too many for most folks) but one can, of
course, just use those that a given product needs.  Personally I like debug
and verbose CLI options in my tools such that if debug is used then Debug level
for output is set, if debug and verbose both are used then Trace level (more
verbose debugging) is set up, and if just Verbose then Verbose level, etc.  I
use a "record" or "logfile" type option for logging if desired and one can use
things like a "quiet" or "silent" option to only see errors or important notes.
Use whatever works for your tool and simply adjust the 'out' package levels to
match.  If your needs are more basic check out Go's "log" and "fmt" packages
for basic output control or something like spf13's jwalterweatherman (jww)
package on github (github.com/spf13/jWalterWeatherman), it's neat and some of
the work here is based on ideas from that (thanks to spf13) and the Go 'log'
package (thanks to the Go authors).

Aside: spf13 also created 'cobra' for easy CLI setup and 'viper' for config file
and environment variable settings/mgmt (ties in nicely w/cobra).  I've modified
'viper' to use this 'out' package instead of the original 'jww' package that
it was using (cobra already uses a writer so no need to tweak it).  Anyhow, see
the 'github.com/dvln' organization for updated copies with tweaks to those if
you wish (keep in mind there are other additions you may or may not want).

The default settings add default "prefixes" on some of the messages, the
defaults for "screen" output are:

1. Trace level (stdout):           "\<date/time\> Trace: \<message\>"
2. Debug level (stdout):           "\<date/time\> Debug: \<message\>"
3. Verbose level (stdout):         "\<message\>""
4. *Default*: Info|Print (stdout): "\<message\>"
5. Note level (stdout):            "Note: \<message\>"
6. Issue level (stdout):           "Issue: \<message\>"
7. Error level (stderr):           "Error: \<message\>"
8. Fatal level \[stack\] (stderr):   "Fatal: \<message\>"

The built-in names for these output levels can't be changed (unless you tweak
the code) but everything visible to the client can be such as if you want the
Issue level to instead print "Warning: " (ie: instead of "Issue: ") that is
not a problem (see SetPrefix).  If you want no prefix or if you want time/date
info turned on/off that's also doable.  If you want everything to go to stdout
or if you want to change the default threshold to Verbose instead of Print/Info,
no problem.  Various adjustments via the API are covered below.

For the default log file output we start with an ioutil.Discard for the
io.Writer which effectively means /dev/null to begin with.  However, if you
set a log file up using provided API's (as below) then the default log file
output will kick on and behave as follows:

1. Trace level:         "\[\<pid\>\] TRACE   \<date/time\> \<shortfile:line#:shortfunc\> Trace: \<message\>"
2. Debug level:         "\[\<pid\>\] DEBUG   \<date/time\> \<shortfile:line#:shortfunc\> Debug: \<message\>"
3. Verbose level:       "\[\<pid\>\] VERBOSE \<date/time\> \<shortfile:line#:shortfunc\> \<message\>"
4. Info|Print level:    "\[\<pid\>\] INFO    \<date/time\> \<shortfile:line#:shortfunc\> \<message\>"
5. Note level:          "\[\<pid\>\] NOTE    \<date/time\> \<shortfile:line#:shortfunc\> Note: \<message\>"
6. Issue level:         "\[\<pid\>\] ISSUE   \<date/time\> \<shortfile:line#:shortfunc\> Issue: \<message\>"
7. Error level:         "\[\<pid\>\] ERROR   \<date/time\> \<shortfile:line#:shortfunc\> Error: \<message\>"
8. Fatal level \[stack\]: "\[\<pid\>\] FATAL   \<date/time\> \<shortfile:line#:shortfunc\> Fatal: \<message\>"

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
