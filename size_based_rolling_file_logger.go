package sherlog

/*
SizeBasedRollingFileLogger is a logger that rolls files when they hit a certain number of log messages.
*/
type SizeBasedRollingFileLogger struct {
	RollingFileLogger
	countToRollOn int
	curCount      int
}

/*
NewRollingFileLoggerWithSizeLimit creates logs that roll when numMessagesPerFile is hit.
*/
func NewRollingFileLoggerWithSizeLimit(logFilePath string, numMessagesPerFile int) (*SizeBasedRollingFileLogger, error) {
	if numMessagesPerFile <= 0 {
		return nil, NewLeveledException("log files must have room for at least 1 message.", EnumError)
	}
	fileLogger, err := NewFileLogger(getTimestampedFileName(logFilePath))
	if err != nil {
		return nil, err
	}
	return &SizeBasedRollingFileLogger{
		RollingFileLogger: RollingFileLogger{
			FileLogger:   *fileLogger,
			baseFilePath: logFilePath,
		},
		countToRollOn: numMessagesPerFile,
	}, nil
}

/*
Log calls loggable's Log function. Is thread safe :)
*/
func (rfl *SizeBasedRollingFileLogger) Log(errToLog error) error {
	err := rfl.RollingFileLogger.Log(errToLog)
	if err != nil {
		return err
	}

	return rfl.incAndRollIfNecessary()
}

/*
LogNoStack calls loggable's LogNoStack function. Is thread safe :)
*/
func (rfl *SizeBasedRollingFileLogger) LogNoStack(errToLog error) error {
	err := rfl.RollingFileLogger.LogNoStack(errToLog)
	if err != nil {
		return err
	}

	return rfl.incAndRollIfNecessary()
}

/*
LogJson calls loggable's LogJson function. Is thread safe :)
*/
func (rfl *SizeBasedRollingFileLogger) LogJson(errToLog error) error {
	err := rfl.RollingFileLogger.LogJson(errToLog)
	if err != nil {
		return err
	}

	return rfl.incAndRollIfNecessary()
}

func (rfl *SizeBasedRollingFileLogger) incAndRollIfNecessary() error {
	rfl.curCount++
	if rfl.curCount >= rfl.countToRollOn {
		return rfl.roll()
	}
	return nil
}

func (rfl *SizeBasedRollingFileLogger) roll() error {
	err := rfl.RollingFileLogger.roll()
	rfl.curCount = 0
	return err
}
