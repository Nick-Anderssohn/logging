package sherlog

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

type logFunction func(writer io.Writer) error

/*
Loggable should be implemented by something for it to be loggable by a Logger's Log function
*/
type Loggable interface {
	error
	Log(writer io.Writer) error
}

/*
LoggableWithNoStackOption should be implemented by something for it to be loggable by a Logger's LogNoStack function
*/
type LoggableWithNoStackOption interface {
	Loggable
	LogNoStack(writer io.Writer) error
}

/*
JsonLoggable should be implemented by something for it to be loggable by a Logger's LogJson function
*/
type JsonLoggable interface {
	error
	LogAsJson(writer io.Writer) error
}

/*
Logger is an interface representing a Logger that can call all of a Loggable's log functions.
*/
type Logger interface {
	Log(errorsToLog ...interface{}) error
	Close()
	LogNoStack(errToLog error) error
	LogJson(errToLog error) error
	Critical(values ...interface{}) error
	Error(values ...interface{}) error
	OpsError(values ...interface{}) error
	Warn(values ...interface{}) error
	Info(values ...interface{}) error
	Debug(values ...interface{}) error
}

/*
FileLogger logs exceptions to a single file path.
Writes are not buffered. Opens and closes per exception written.
*/
type FileLogger struct {
	logFilePath string
	mutex       *sync.Mutex
	file        *os.File
}

/*
NewFileLogger create a new FileLogger that will write to logFilePath. Will append to the file if it already exists. Will
create it if it doesn't.
*/
func NewFileLogger(logFilePath string) (*FileLogger, error) {
	file, err := openFile(logFilePath)
	if err != nil {
		return nil, AsError(err)
	}

	return &FileLogger{
		logFilePath: logFilePath,
		file:        file,
		mutex:       new(sync.Mutex),
	}, nil
}

func openFile(fileName string) (*os.File, error) {
	return os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
}

/*
Log calls loggable's Log function. Is thread safe :)
Non-sherlog errors get logged with only timestamp and message
*/
func (l *FileLogger) Log(errorsToLog ...interface{}) error {
	if len(errorsToLog) < 1 {
		return AsError("no parameters provided to Log")
	}

	l.mutex.Lock()
	defer func() {
		l.file.Write([]byte("\n\n"))
		l.mutex.Unlock()
	}()
	for i, errToLog := range errorsToLog {
		if errToLog == nil {
			return AsError("tried to log nil error")
		}

		switch impl := errToLog.(type) {
		case Loggable:
			err := l.log(impl.Log)
			if err != nil {
				return AsError(err)
			}
		case error:
			err := l.logNonSherlogError(impl)
			if err != nil {
				return AsError(err)
			}
		default:
			l.file.Write([]byte(fmt.Sprintf("%v", impl)))
		}

		if i < len(errorsToLog)-1 {
			l.file.Write([]byte("\nCaused by:\n"))
		}
	}
	return nil
}

/*
LogNoStack calls loggable's LogNoStack function. Is thread safe :)
Non-sherlog errors get logged with only timestamp and message
*/
func (l *FileLogger) LogNoStack(errToLog error) error {
	if errToLog == nil {
		return AsError("tried to log nil error")
	}

	l.mutex.Lock()
	defer func() {
		l.file.Write([]byte("\n\n"))
		l.mutex.Unlock()
	}()

	if loggable, isLoggable := errToLog.(LoggableWithNoStackOption); isLoggable {
		return l.log(loggable.LogNoStack)
	}
	return l.logNonSherlogError(errToLog)
}

/*
LogJson calls loggable's LogJson function. Is thread safe :)
Non-sherlog errors get logged with only timestamp and message
*/
func (l *FileLogger) LogJson(errToLog error) error {
	if errToLog == nil {
		return AsError("tried to log nil error")
	}

	l.mutex.Lock()
	defer func() {
		l.file.Write([]byte("\n"))
		l.mutex.Unlock()
	}()

	if loggable, isLoggable := errToLog.(JsonLoggable); isLoggable {
		return l.log(loggable.LogAsJson)
	}

	// Else, manually extract info...
	jsonBytes, err := json.Marshal(map[string]interface{}{
		"Time":    time.Now().In(Location).Format(timeFmt), // Use log time instead of time of creation since we don't have one....
		"Message": errToLog.Error(),
	})
	if err != nil {
		return err
	}

	_, err = l.file.Write(jsonBytes)
	return err
}

/*
Close closes the file writer.
*/
func (l *FileLogger) Close() {
	l.file.Close()
}

func (l *FileLogger) log(logFunc logFunction) error {
	err := logFunc(l.file)
	if err != nil {
		return err
	}
	//l.file.Write([]byte("\n\n"))
	err = l.file.Sync() // To improve perf, may want to move this to just run every minute or so
	if err != nil {
		return err
	}
	return nil
}

func (l *FileLogger) logNonSherlogError(errToLog error) error {
	now := time.Now().In(Location).Format(timeFmt) // Use log time instead of time of creation since we don't have one....

	_, err := l.file.Write([]byte(now))
	if err != nil {
		return err
	}

	_, err = l.file.Write([]byte(" - "))
	if err != nil {
		return err
	}

	_, err = l.file.Write([]byte(errToLog.Error()))
	return err
}

/*
Critical turns values into a *LeveledException with level CRITICAL and then calls the logger's
Log function.
*/
func (l *FileLogger) Critical(values ...interface{}) error {
	return l.Log(graduateOrConcatAndCreate(EnumCritical, values...))
}

/*
Error turns values into a *LeveledException with level ERROR and then calls the logger's
Log function.
*/
func (l *FileLogger) Error(values ...interface{}) error {
	return l.Log(graduateOrConcatAndCreate(EnumError, values...))
}

/*
OpsError turns values into a *LeveledException with level OPS_ERROR and then calls the logger's
Log function.
*/
func (l *FileLogger) OpsError(values ...interface{}) error {
	return l.Log(graduateOrConcatAndCreate(EnumOpsError, values...))
}

/*
Warn turns values into a *LeveledException with level WARNING and then calls the logger's
Log function.
*/
func (l *FileLogger) Warn(values ...interface{}) error {
	return l.Log(graduateOrConcatAndCreate(EnumWarning, values...))
}

/*
Info turns values into a *LeveledException with level INFO and then calls the logger's
Log function.
*/
func (l *FileLogger) Info(values ...interface{}) error {
	return l.Log(graduateOrConcatAndCreate(EnumInfo, values...))
}

/*
Debug turns values into a *LeveledException with level DEBUG and then calls the logger's
Log function.
*/
func (l *FileLogger) Debug(values ...interface{}) error {
	return l.Log(graduateOrConcatAndCreate(EnumDebug, values...))
}
