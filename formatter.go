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

// Formatter (interface) gives another way to control what is dumped to the
// screen and log file (beyond prefixes/flags).  One can augment/convert or
// even suppress output as desired.  With this one can set any output format
// desired or push info elsewhere and suppress output.  Formatter fires once
// on output without any flags applied (but prefixes can have been applied),
// the following comes into a formatter:
// - msg: this is the raw message before markup (can have basic prefixes on msg)
// - level: log level (Trace, Debug, Verbose, Info/Print, Note, Issue, Error, Fatal)
// - code: msg or error code (if available, if not a default setting)
// - stack: stack trace if one is available
// - dying: boolean True if fatal situation (only Issue/Error/Fatal levels)
// The return data is essentially:
// - msg (string): the update message or just the passed in msg if no updates
// - supressOut (int): 0 if not suppressing any output otherwise, if set, one
// - supressNativePrefixing (bool): timestamps and prefixes are still applied
// to the result of a Formatter unless this is set to true, then not applied
// would set it to ForScreen, ForLogfile or ForBoth if one wishes to forcibly
// suppress the output to those targets (eg: maybe you stored the error
// elsewhere in some JSON package and want it included in the final JSON out
// for instance and not now).
//
// Example 1: you have used 'out' API's to supress all prefixes and flags
// formatting completely for screen and log output... you want to do your own
// formatting and keep it identical for screen and log output.  Go for it!
// Take the message and log level info and put anything you want in with it
// and return your new message... it will then be dumped by the 'out' pkg to
// screen and logfile as configured (it can also be augmented by out flags if
// you have left them on... independently for screen/log output).
//
// Example 2: a CLI tool can output text (normal) but also has a JSON output
// mode.  For issue/error/fatal msgs (terminal or not) I want to get that
// data into my JSON structure so I can present it cleanly in my single JSON
// output structure.  If it's non-fatal I'll push the non-fatals into my JSON
// module so it records em and I'll return "" so they don't go to screen/log
// until the full JSON output is dumped, it'll have a 'warnings' key there with
// an array of warnings (or whatever) along with the rest of my JSON struct.
// I have a fatal error... doh!, adjust the message into a JSON format on the
// fly *right now* in my formatter and return that... 'out' will dump it to
// the screen and log file for me in a JSON parsable way, done (note that my
// flags will be honored so logfile can have usual prefixes and meta-data as
// can screen, if desired, just like all output coming through 'out'... if you
// don't want it you can see when your tool is in JSON mode and flip that stuff
// all off, no problem, well before getting here).
type Formatter interface {
	// This returns the error message without the stack trace.
	FormatMessage(msg string, outLevel Level, code int, stack string, dying bool) (string, int, bool)
}

// SetFormatter sets a formatter against the output package so that output
// can be pre-formatted (or cleared) before being dumped to the screen/logfile,
// see the description of the Formatter interface.
func SetFormatter(level Level, formatter Formatter) {
	for _, o := range outputters {
		o.mu.Lock()
		defer o.mu.Unlock()
		if level == LevelAll || o.level == level {
			o.formatter = formatter
			if o.level == level {
				break
			}
		}
	}
}

// ClearFormatter clears the formatters on a given level or all levels
// if the LevelAll level is used.
func ClearFormatter(level Level) {
	for _, o := range outputters {
		o.mu.Lock()
		defer o.mu.Unlock()
		if level == LevelAll || o.level == level {
			o.formatter = nil
			if o.level == level {
				break
			}
		}
	}
}
