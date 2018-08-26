package sherlog

import (
	"log"
	"sync"
)

/*
A simple container for multiple loggers.
Will call all of the loggers' log functions every time something
needs to be logged.
 */
type PolyLogger struct {
	Loggers          []Logger
	handleLoggerFail func(error)
	waitGroup        sync.WaitGroup
}

/*
loggers are all the loggers that will be used during logging. If a logger fails when
logging something, log.Println will be used to log the error that the logger returned.
Returns a new PolyLogger.
 */
func NewPolyLogger(loggers []Logger) *PolyLogger {
	return NewPolyLoggerWithHandleLoggerFail(loggers, defaultHandleLoggerFail)
}

/*
loggers are all the loggers that will be used during logging. handleLoggerFail is run whenever
one of those loggers returns an error while logging something (indicating that it failed to log the message).
Returns a new PolyLogger
 */
func NewPolyLoggerWithHandleLoggerFail(loggers []Logger, handleLoggerFail func(error)) *PolyLogger {
	return &PolyLogger{
		Loggers: loggers,
		handleLoggerFail: handleLoggerFail,
	}
}

/*
Asynchronously runs all loggers' Close functions.
 */
func (p *PolyLogger) Close() {
	for _, logger := range p.Loggers {
		go logger.Close()
	}
}

/*
Asynchronously runs all logger's Log functions.
Handles any errors in the logging process with handleLoggerFail.
Will always return nil.
 */
func (p *PolyLogger) Log(errToLog error) error {
	for _, logger := range p.Loggers {
		p.waitGroup.Add(1)
		go p.runLoggerWithFail(logger.Log, errToLog)
	}
	p.waitGroup.Wait()
	return nil
}

/*
Asynchronously runs all logger's LogNoStack functions.
Will ignore any Loggers that are not RobustLoggers.
Handles any errors in the logging process with handleLoggerFail.
Will always return nil.
 */
func (p *PolyLogger) LogNoStack(errToLog error) error {
	for _, logger := range p.Loggers {
		if robustLogger, isRobust := logger.(RobustLogger); isRobust {
			p.waitGroup.Add(1)
			go p.runLoggerWithFail(robustLogger.LogNoStack, errToLog)
		}
	}
	p.waitGroup.Wait()
	return nil
}

/*
Asynchronously runs all logger's LogJson functions.
Will ignore any Loggers that are not RobustLoggers.
Handles any errors in the logging process with handleLoggerFail.
Will always return nil.
 */
func (p *PolyLogger) LogJson(errToLog error) error {
	for _, logger := range p.Loggers {
		if robustLogger, isRobust := logger.(RobustLogger); isRobust {
			p.waitGroup.Add(1)
			go p.runLoggerWithFail(robustLogger.LogJson, errToLog)
		}
	}
	p.waitGroup.Wait()
	return nil
}

// Call in a go routine! Will automatically decrement wait group
func (p *PolyLogger) runLoggerWithFail(logFunc func(error) error, loggable error) {
	defer p.waitGroup.Add(-1)
	err := logFunc(loggable)
	if err != nil && p.handleLoggerFail != nil {
		p.handleLoggerFail(err)
	}
}

func defaultHandleLoggerFail(err error) {
	log.Println(err)
}