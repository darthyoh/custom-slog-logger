# custom-slog-logger
Custom logger based on slog used to display nice colored messages on any **io.Writer** including source lines of error. In addition, the logger can be configured to send JSON formatted log to a third-party http server (e.g. a logging microservice)

As a **slog.Logger** using a custom **slog.Handler**, it can be used to generate new logger, with additinonal attributes that will be print each time a **slog.Record** has to be handled (e.g. an user id in the context of a http server), or with a group name (i.e. prefix for the attributes), or both of them.

This logger is customisable with the use of **CustomHandlerOptions**.

## Installation

A simple `go get github.com/darthyoh/custom-slog-logger` should install the package in your module.

## Basic usage

Generate a simple **CustomLogger** with **NewCustomLogger()** utility and start logging :

```
package main

import customlogger "github.com/darthyoh/curstom-slog-logger"

func main() {

	//create a new logger with default (nil) *CustomHandlerOptions
	//will print colored log on os.Stderr, with source code and a default Info level
	//no json url server provided
	logger := customlogger.NewCustomLogger(os.Stderr, nil)

	//simple Info log
	logger.Info("using custom logger", "an attr", "a value")

	//simple Error log
	logger.Error("fatal error", "error_message","the error message")

	//simple Debug log
	logger.Debug("test debug") //won't be printed : Debug Level in not enough for this logger !
	
	//simply log to os.Stderr
	logger.InfoTextOnly("Info log")

	//simply send to server
	logger.InfoJsonOnly("Info log") //won't be printed but WON'T BE SENT, cause no jsonurl was provided
}
```

