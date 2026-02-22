package commonGo

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	logger "github.com/multiversx/mx-chain-logger-go"
	"github.com/multiversx/mx-chain-logger-go/file"
)

// AttachFileLogger attaches, if required, a log file
func AttachFileLogger(
	log logger.Logger,
	defaultLogsPath string,
	logFilePrefix string,
	saveLogFile bool,
	workingDir string) (FileLoggingHandler, error) {
	var err error
	var logFile FileLoggingHandler
	if saveLogFile {
		argsFileLogging := file.ArgsFileLogging{
			WorkingDir:      workingDir,
			DefaultLogsPath: defaultLogsPath,
			LogFilePrefix:   logFilePrefix,
		}
		logFile, err = file.NewFileLogging(argsFileLogging)
		if err != nil {
			return nil, fmt.Errorf("%w creating a log file", err)
		}
	}

	err = logger.SetDisplayByteSlice(logger.ToHex)
	log.LogIfError(err)

	return logFile, nil
}

// ReadEnvFile will read the file contents in the provided map
func ReadEnvFile(envFile string, m map[string]string) error {
	err := godotenv.Load(envFile)
	if err != nil {
		return err
	}

	for k := range m {
		val := os.Getenv(k)
		if len(val) == 0 {
			return fmt.Errorf("%s is not set in the .env file", k)
		}

		m[k] = val
	}

	return nil
}

// CronJobStarter is able to start a go routine that periodically calls the provided handler. The time between calls is
// provided as timeToCall
func CronJobStarter(ctx context.Context, handler func(ctx context.Context), timeToCall time.Duration) {
	go func() {
		timer := time.NewTimer(timeToCall)
		defer timer.Stop()

		handler(ctx)

		for {
			select {
			case <-timer.C:
				handler(ctx)
				timer.Reset(timeToCall)
			case <-ctx.Done():
				return
			}
		}
	}()
}
