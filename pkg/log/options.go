/*
 * Copyright 2021 The KunStack Authors.
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 * http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package log

import (
	"github.com/spf13/pflag"
)

// Options log related configuration
type Options struct {
	// log level, optional value trace,info,warn,error,panic,fatal
	Level string `yaml:"level,omitempty" json:"level,omitempty"`
	// log file path
	File string `yaml:"file,omitempty" json:"file,omitempty"`
	// time encoder eg: RFC3339 , RFC3339NANO, RFC822, RFC850, RFC1123, STAMP
	Time string `yaml:"time,omitempty" json:"time,omitempty"`
	// caller encoder optional: long, short
	Caller string `yaml:"caller,omitempty" json:"caller,omitempty"`
	// log encoder, optional: text, json
	Format string `yaml:"format,omitempty" json:"format,omitempty"`
}

// SetDefaults sets the default values.
func (l *Options) SetDefaults() {
	l.Time = "RFC3339"
	l.Format = "TEXT"
	l.Level = "INFO"
	l.Caller = "NONE"
}

// Flags Returns a collection of command line flags
func (l *Options) Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("log", pflag.ContinueOnError)
	fs.StringVar(&l.Level, "log.level", l.Level, "Log level,Those below this level will not be output optional value trace,info,warn,error,panic,fatal")
	fs.StringVar(&l.File, "log.file", l.File, "The path of the log file, if the file does not exist, it will be created automatically")
	fs.StringVar(&l.Time, "log.time", l.Time, "The encoder of the time field in the log, eg: RFC3339, RFC3339NANO, RFC822, RFC850, RFC1123, STAMP")
	fs.StringVar(&l.Caller, "log.caller", l.Caller, "The log call encoder specifies how the log line number is displayed in the source file, eg: long, short")
	fs.StringVar(&l.Format, "log.format", l.Format, "Log format encoder, specify the display format of the log, eg: text, json")
	return fs
}

// Validate verify the configuration and return an error if correct
func (l *Options) Validate() error {
	_, err := ParseLevel(l.Level)
	if err != nil {
		return err
	}
	_, err = ParseCaller(l.Caller)
	if err != nil {
		return err
	}
	_, err = ParseEncoder(l.Format)
	if err != nil {
		return err
	}
	return nil
}

// NewOptions Create an Options filled with default values
func NewOptions() *Options {
	opt := &Options{}
	opt.SetDefaults()
	return opt
}
