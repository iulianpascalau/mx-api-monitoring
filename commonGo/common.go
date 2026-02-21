package commonGo

import (
	"fmt"
	"os"

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
