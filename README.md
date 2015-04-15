out
===

Package for "leveled" CLI output printing to a terminal/screen with easy
mirroring of that output to a log file (all output via Go io.Writer's).
The goal is to be as easy to use as fmt.Printf, for example, but with
various levels, eg: out.Printf, out.Println, out.Print, out.Debugln,
out.Issueln, out.Issuef, out.Errorln, out.Errorf, out.Fatal, out.Fatalf, etc.
Combine this with independent control of prefixes, flags for time/date and
file/line#, output thresholds and clean indenting and prefixing for multi-line
messages as well as for non-newline terminated messages.

This started life as a wrapper around the Go standard 'log' library allowing two
Loggers to be independently controlled (one typically writing to the screen and
one to a log file), based on spf13's jwalterweatherman package... but upon
further use I found 'log' to be too limiting and so built everything in using
io.Writer's directly.  Features:

1. Ready for basic screen output immediately
2. Trivial to "turn on" logging to a for a temp or named log file
3. Independent io.Writer control over screen and log file output thresholds
4. Independent control over flags (ie: add metadata like date/time, file)
5. Clean alignment and handling for multi-line strings or strings w/no newlines
6. Avoids insertion of newlines into the screen or log file io.Writers
7. Ability to easily add stack traces on Fatal class errors

# Usage

## Basic usage:
Put calls throughout your code based on desired "levels" of messaging.
Simply run calls to the output functions:

 * out.Trace\[f|ln\](...)  
 * out.Debug\[f|ln\](...)
 * out.Verbose\[f|ln\](...)
 * out.Print\[f|ln\](...) || out.Info\[f|ln\](...)       (same)
 * out.Note\[f|ln\](...)
 * out.Issue\[f|ln\](...)
 * out.Error\[f|ln\](...)
 * out.Fatal\[f|ln\](...)

Each of these map to two io.Writers, one for the screen and one for the
log file (default is to discard log file output until it is configured,
see below).  Standard usage:

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
        // Something like an issue to take note of but recoverable
        out.Issueln(err, "\nRecovering and continuing ..")
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

There are 8 levels but you can use just those that your product needs
of course.  Personally I like to have at least a --debug and --verbose
set of options in my CLI's (with -d and -v short names) so that items
like "-dv" can map to Trace level output, "-d" to Debug level output
and "-v" to Verbose level output with none of these opts mapping to
normal Print/Info level output.  Extend with "--terse" and "-t" and/or
a "--quiet" and "-q" or other options to map to higher levels of output
along with items like "--log" to kick on temp file logging.  Do what you
wish of course.

The default settings add prefixes for you on some of the messages, the
defaults are:

1. Trace level (stdout): "\<date/time\> Trace: message"
2. Debug level (stdout): "\<date/time\> Debug: message"
3. Verbose level (stdout): message
4. *Default*: Info|Print level (stdout): "message"
5. Note level (stdout): "Note: message"
6. Issue level (stdout): "Issue: message"
7. Error level (stderr): "ERROR: message"
8. Fatal level (stderr), optional stacktrace: "FATAL: message""

The level names can't be changed unless you tweak the code but
everything else can be such as time/date or not, any prefix to
be output to the screen or log file for a given level, what
io.Writer is used for all levels or a specific level, etc,
see API's below.

If you don't like the markup/prefix or the screen output "default" 
threshold that is all changable via the API.  If you do want to
see trace or debug or verbose output then change the default screen
output threshold (see below).  Until that is done anything "above"
your logging level on this list will to to /dev/null effectively.

For log files keep in mind the default is ioutil.Discard for the
io.Writer effectively meaning /dev/null for output, but if you set
a file logger up (as below) the default output looks like this (you
must also tell it what the default level is when you set it up):

1. Trace level: "date/time file/line# Trace: message"
2. Debug level: "date/time file/line# Debug: message"
3. Verbose level: "date/time file/line# message"
4. Info|Print level: "date/time file/line# message"
5. Note level: "date/time file/line# Note: message"
6. Issue level: "date/time file/line# Issue: message"
7. Error level: "date/time file/line# ERROR: message"
8. Fatal level, optional stacktrace: "date/time file/line# FATAL: message"

Again, all of this is adjustable, see the next section.  To activate
a log you need to do two things:

1. Use an API call to prepare a temp or named file (points io.Writer at it)
2. Give the logging level for output you want to go into the log file

Lets see how to configure and use this package next.

## Step 2. Optionally configure 'out' package

To set up file logging or to adjust any of the defaults listed above
follow these steps:

### Using a temp log file, setting detailed output/logging thresholds

Here we'll enable all available levels of output to both the screen and to
the temp log file:

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

After that all out package output functions (eg: out.Debugf or out.Println)
will go to the screen io.Writer and to the log file io.Writer using the default
settings (unless you have adjusted those yourself as below).

### Adjust the screen verbosity so we see verbose output only

You should call this early in your application as only
calls after it is done will use the set output level:

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
on any log file output stream (if one is set up).

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

This is a first foray into Go... I liked spf13's jwalterweatherman module
but I wanted a bit more independent control over the flags for each logger
and I found the 'log' packages handling of output formatting, multi-line
strings and strings with no newlines to not quite behave as I wanted (aside:
spf13's module uses Go's log package, this package started that way also).

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
Note: Successful test of: somesystem
Note: So I think you should
Note: buy this system as it
Note: is a quality system
```

If it was Debug instead of Note the output would be:

```text
date/time Debug: Successful test of: somesystem
date/time Debug: So I think you should
date/time Debug: buy this system as it
date/time Debug: is a quality system
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
	out.SetFlag(LevelAll, 0, ForLogfile)
	out.SetFlag(LevelTrace, out.Lstdflags, ForLogfile)
	out.SetFlag(LevelDebug, out.Lstdflags, ForLogfile)
    ...
    // Now lets set up a buffer for the "logfile" output (it is, of course,
    // no longer a log file really, it's a buffer... but to this package the
    // io.Writer I'm mucking with is the "log file" writer:
	logfileBuf := new(bytes.Buffer)
	out.SetWriter(logfileBuf, out.ForLogfile)
    ...
```

# Current status
This is a very early release... needs more work so use with caution (vendor)
as it's likely to change a bit over the coming months.  Thanks again to spf13
and the Go authors for some ideas here and definitely feel free to send in any
issues.

I wrote this for use in [dvln](http://github.com/dvln). Yeah, doesn't really
exist as much yet but we shall see if I can change that.  It's targeted towards
nested multi-pkg multi-scm development line and workspace management... what
could go wrong?  ;)
