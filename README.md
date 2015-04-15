out
===

Easy to use "leveled" CLI output printing to the terminal/screen with easy mirroring
of that output to a file log (both streams via an io.Writer).  As easy to use as
fmt.Print(), fmt.Println(), fmt.Printf() via out.Print(), out.Println(), out.Printf()
as well as various other levels, eg: out.Traceln(), out.Debugln(), out.Verboseln(),
out.Noteln(), out.Issueln(), out.Errorln(), out,Fatalln().

This started life as a wrapper around the Go standard 'log' library allowing two
io.Writers to be independently controlled (one typically writing to the screen and
one to a log file) but upon investigation the package needed to stop using 'log'
to get all the features desired:

1. Ready to go out of the box. 
2. One library for both printing to the screen/terminal and logging (to a file).
3. Really easy to log to either a temp file or a file you specify.
4. Ability to easily add stack traces on fatal class errors
5. Independent control over screen output level and logging output level
6. Independent control over flags for the different output streams
7. Better alignment and handling for multi-line strings or strings w/no newlines

# Usage

## Step 1. Use it
Put calls throughout your source based on type of feedback.
No initialization or setup needs to happen. Just start calling things.

Available screen/logfile output available via:

 * out.Trace[f|ln](...)
 * out.Debug[f|ln](...)
 * out.Verbose[f|ln](...)
 * out.Print[f|ln](...)    ||   out.Info[f|ln](...)       [identical]
 * out.Note[f|ln](...)
 * out.Issue[f|ln](...)
 * out.Error[f|ln](...)
 * out.Fatal[f|ln](...)

Each of these map to two io.Writers, one for the screen and one for the
logfile (default is ioutils.Discard io.Writer for the logfile but one can
set it up trivially as below).  Standard usage:

```go
    import (
        "github.com/dvln/out"
    )

    // Day to day normal output
    out.Printf("some info: %s\n", infostr)

    // Less important output not shown by default
    out.Verbosef("data was pulled from src: %s\n", source)

    ...
    if err != nil {
        // This is a user error, called an issue currently
        out.Issueln(err)
    }
    ...
    if err2 != nil {
        // This is a fatal error, I want a stack trace too it's so bad
		out.SetStacktraceOnExit(true)
        out.Fatalln(err2)
    }

```

Use only whatever levels you want and adjust output thresholds as you
like.  I like my CLI to have a --debug and --verbose set of options
ideally also a -d and -v corresponding to those.  With those two one
can set -dv for a Trace type output (extreme detail w/extreme debug),
or a -d for Debug level output without detailed Trace debug level,
or just -v for Verbose level output and everything below that... or
nothing for basic Print level output.  If you want a --quiet | -q
it might map to the Issue level or Error level and below... with
perhaps a --terse | -t mapping to the Note level or Issue level.
Up to you.

The default settings add prefixes for you on some of the messages, the
defaults are:

1. Trace level (stdout):      <date/time> Trace: <msg>
2. Debug level (stdout):      <date/time> Debug: <msg>
3. Verbose level (stdout):    <msg>
4. Info|Print level (stdout): <msg>           <--- Default Threshold for Screen
5. Note level (stdout):       Note: <msg>
6. Issue level (stdout):      Issue: <msg>
7. Error level (stderr):      ERROR: <msg>
8. Fatal level (stderr):      FATAL: <msg>
                     [optional stacktrace]

If you don't like the markup/prefix or the screen output "default" 
threshold that is all changable via the API.  If you do want to
see trace or debug or verbose output then change the default screen
output threshold (see below).  Until that is done anything "above"
your logging level on this list will to to /dev/null effectively.

For log files, if you activate one, the defaults are:

1. Trace level:      <date/time> <file/line#> Trace: <msg>
2. Debug level:      <date/time> <file/line#> Debug: <msg>
3. Verbose level:    <date/time> <file/line#> <msg>
4. Info|Print level: <date/time> <file/line#> <msg>
5. Note level:       <date/time> <file/line#> Note: <msg>
6. Issue level:      <date/time> <file/line#> Issue: <msg>
7. Error level:      <date/time> <file/line#> ERROR: <msg>
8. Fatal level:      <date/time> <file/line#> FATAL: <msg>
                     [optional stacktrace]

Again, all of this is adjustable (see next section).  To activate
a log you need to do two things:

1. Use an API call to log to a tmp or named file
2. Identify the logging level you want to put in the log file,
   the default is to discard logging so set it at out.LevelInfo,
   see next section

## Step 2. Optionally configure 'out' package

See defaults above... to adjust those items you've come to the
right section.

### Using a temp logfile, setting detailed output/logging thresholds

If you want to enable all levels of output to the screen and to
a temp logfile you might do something like this:

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
    // and if we are in verbose debugging mode lets set up tracing
    if Debug && Verbose {
        // lets set both screen and logfile to Trace level:
        out.SetThreshold(out.LevelTrace, out.ForBoth)
    }
```

### Just adjust the screen verbosity to see verbose output

JWW conveniently creates a temporary file and sets the log Handle to
a io.Writer created for it. You should call this early in your application
initialization routine as it will only log calls made after it is executed. 
When this option is used, the library will fmt.Println where to find the
log file.

```go
    import (
        "github.com/dvln/out"
    )

    if Verbose {
        // set the threshold level for the screen to verbose output
        out.SetThreshold(out.LevelVerbose, out.ForScreen)
    }
```

### Screen output managed by the defaults but log to given file at Debug level

In this case we'll use another API to set up the logfile io.Writer to
a specific file:

```go
    import (
        "github.com/dvln/out"
    )
    ...
    out.SetLogFile("/some/dir/logfile")
    out.SetThreshold(out.LevelDebug, out.ForLogfile)
```

### Simple usage example showing some formatting

This is a first foray into Go... I liked spf13's jwalterweatherman module
but I wanted a bit more independent control over the flags for each logger
and I found the 'log' packages handling of multi-line strings and strings
with no newlines to not be acceptable for screen output where data might
need to be collated.  I also preferred the prefix as it relates to the
levels to be on the right of the timestamp and not the left, eg:

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

```Note: Successful test of: <somesystem>
Note: So I think you should
Note: buy this system as it
Note: is a quality system
```

And if it was Debug instead of Note the output would be:

```<date/time> Debug: Successful test of: <somesystem>
<date/time> Debug: So I think you should
<date/time> Debug: buy this system as it
<date/time> Debug: is a quality system
```

### Adding in short filename and line#, for screen debug level:

```go
    import (
        "github.com/dvln/out"
    )
    ...
    // Note that out.Lstdflags is out.Ldate|out.Ltime (date/time)
	out.SetFlag(LevelDebug, out.Lstdflags|out.Lshortfile, ForScreen)
    ...

    out.Debugln("Successful test of: ")
    systemx := grabTestSystem()
    out.Debugf("%s\n", systemx)
	out.Debug("So I think you should\nbuy this system as it\nis a quality system\n")
```

### Replace the screen output so it instead goes into a buffer

Lets switch the io.Writer used for the screen so it goes into a byte buffer:

```go
    import (
        "github.com/dvln/out"
    )
	screenBuf := new(bytes.Buffer)
	out.SetWriter(screenBuf, out.ForScreen)
    ...
```

Or perhaps better yet we leave the screen alone and use the logfile as
a buffer instead:

```go
    import (
        "github.com/dvln/out"
    )

    // First let me turn off logfile defaults so they look more
    // like screen defaults (only timestamps for Trace/Debug, otherwise
	// no timestamps or filenames/etc added in), note that the LevelAll
    // impacts all levels whereas specific levels only impact those
    // levels (right now it's not a bit flag so it's all or one):
	out.SetFlag(LevelAll, 0, ForLogfile)
	out.SetFlag(LevelTrace, out.Lstdflags, ForLogfile)
	out.SetFlag(LevelDebug, out.Lstdflags, ForLogfile)
    ...
    // Now lets set up a buffer for the "logfile" output (which it
    // no longer is of course, it's bufferred output but it's still
    // known as the logfile stream/io.Writer within this pkg):
	logfileBuf := new(bytes.Buffer)
	out.SetWriter(logfileBuf, out.ForLogfile)
    ...
```

# Current status
This is a very early release... needs more work so use with caution (vendor)
as it's likely to change a bit over the coming months.

I wrote this for use in [dvln](http://github.com/dvln). Doesn't exist yet but
we shall see if I can change that.  It's targeted at multi-pkg multi-scm 
development line and workspace management.
