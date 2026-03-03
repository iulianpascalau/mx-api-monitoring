package factory

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/iulianpascalau/api-monitoring/commonGo"
	"github.com/iulianpascalau/api-monitoring/services/aggregation/alarm"
	"github.com/iulianpascalau/api-monitoring/services/aggregation/alarm/executors"
	"github.com/iulianpascalau/api-monitoring/services/aggregation/alarm/notifiers"
	"github.com/iulianpascalau/api-monitoring/services/aggregation/api"
	"github.com/iulianpascalau/api-monitoring/services/aggregation/common"
	"github.com/iulianpascalau/api-monitoring/services/aggregation/config"
	"github.com/iulianpascalau/api-monitoring/services/aggregation/storage"
	"github.com/multiversx/mx-chain-core-go/core/check"
	logger "github.com/multiversx/mx-chain-logger-go"
	"github.com/multiversx/mx-sdk-go/core/polling"
)

const unknownWeekDay = -2

var log = logger.GetOrCreate("factory")

type componentsHandler struct {
	store                 api.Storage
	server                Server
	notifiers             []executors.Notifier
	pollingHandlerTrigger PollingHandler
	statusHandler         alarm.StatusHandler
	alarmService          AlarmEngine
}

// NewComponentsHandler creates a new components handler
func NewComponentsHandler(
	sqlitePath string,
	envFileContents map[string]*commonGo.EnvValue,
	cfg config.Config,
	notifyLogger logger.Logger,
	appVersion string,
) (*componentsHandler, error) {
	store, err := storage.NewSQLiteStorage(sqlitePath, cfg.RetentionSeconds)
	if err != nil {
		return nil, err
	}

	serverArgs := api.ArgsWebServer{
		ServiceKeyApi:             envFileContents[common.EnvServiceKey].Value,
		AuthUsername:              envFileContents[common.EnvAuthUser].Value,
		AuthPassword:              envFileContents[common.EnvAuthPassword].Value,
		ListenAddress:             cfg.ListenAddress,
		StaticDir:                 cfg.StaticDir,
		Storage:                   store,
		GeneralHandler:            api.CORSMiddleware,
		NumSecondsToConsiderStale: cfg.NumSecondsToConsiderStale,
		AppVersion:                appVersion,
	}

	server, err := api.NewServer(serverArgs)
	if err != nil {
		return nil, err
	}

	components := &componentsHandler{
		store:  store,
		server: server,
	}

	err = components.addAlarmComponents(envFileContents, cfg, notifyLogger, store)
	if err != nil {
		return nil, err
	}

	return components, nil
}

func (ch *componentsHandler) addAlarmComponents(
	envFileContents map[string]*commonGo.EnvValue,
	cfg config.Config,
	notifyLogger logger.Logger,
	store alarm.Storage,
) error {
	if !cfg.Alarms.Enabled {
		return nil
	}

	var err error

	ch.notifiers, err = buildNotifiers(notifyLogger, envFileContents, cfg)
	if err != nil {
		return err
	}

	argsNotifiersHandler := executors.ArgsNotifiersHandler{
		Notifiers:          ch.notifiers,
		NumRetries:         cfg.Alarms.NumRetries,
		TimeBetweenRetries: time.Duration(cfg.Alarms.SecondsBetweenRetries) * time.Second,
	}
	notifiersHandler, err := executors.NewNotifiersHandler(argsNotifiersHandler)
	if err != nil {
		return err
	}

	ch.statusHandler, err = executors.NewStatusHandler(notifiersHandler)
	if err != nil {
		return err
	}

	if cfg.Alarms.NumSecondsLoopTimeAlarm < 1 {
		return fmt.Errorf("invalid value for NumSecondsLoopTimeAlarm: %v", cfg.Alarms.NumSecondsLoopTimeAlarm)
	}

	loopTimeAlarmService := time.Duration(cfg.Alarms.NumSecondsLoopTimeAlarm) * time.Second
	ch.alarmService, err = alarm.NewAlarmService(
		store,
		notifiersHandler,
		ch.statusHandler,
		uint32(cfg.NumSecondsToConsiderStale),
		loopTimeAlarmService,
	)
	if err != nil {
		return err
	}

	return ch.addSelfCheckAlarmComponents(cfg)
}

func (ch *componentsHandler) addSelfCheckAlarmComponents(cfg config.Config) error {
	if !cfg.Alarms.SystemSelfCheck.Enabled {
		return nil
	}

	dayOfWeek, err := parseWeekday(cfg.Alarms.SystemSelfCheck.DayOfWeek)
	if err != nil {
		return err
	}

	argsStatusHandlerTrigger := executors.ArgsStatusHandlerTrigger{
		TimeFunc:      time.Now,
		Executor:      ch.statusHandler,
		TriggerDay:    dayOfWeek,
		TriggerHour:   cfg.Alarms.SystemSelfCheck.Hour,
		TriggerMinute: cfg.Alarms.SystemSelfCheck.Minute,
	}
	statusHandlerTrigger, err := executors.NewStatusHandlerTrigger(argsStatusHandlerTrigger)
	if err != nil {
		return err
	}

	argsPollingHandlerTrigger := polling.ArgsPollingHandler{
		Log:              log,
		Name:             "",
		PollingInterval:  time.Second * time.Duration(cfg.Alarms.SystemSelfCheck.PollingIntervalInSec),
		PollingWhenError: time.Second * time.Duration(cfg.Alarms.SystemSelfCheck.PollingIntervalInSec),
		Executor:         statusHandlerTrigger,
	}
	ch.pollingHandlerTrigger, err = polling.NewPollingHandler(argsPollingHandlerTrigger)

	return err
}

func buildNotifiers(notifyLogger logger.Logger, envFileContents map[string]*commonGo.EnvValue, cfg config.Config) ([]executors.Notifier, error) {
	notifiersCollection := make([]executors.Notifier, 0, 10)

	if !check.IfNil(notifyLogger) {
		notifier, _ := notifiers.NewLogNotifier(notifyLogger)
		notifiersCollection = append(notifiersCollection, notifier)
		log.Debug("enabled log notifier")
	}

	pushoverToken := envFileContents[common.EnvPushoverToken].Value
	pushoverUserkey := envFileContents[common.EnvPushoverUserKey].Value
	if len(pushoverToken) > 0 && len(pushoverUserkey) > 0 {
		notifier := notifiers.NewPushoverNotifier(cfg.Alarms.PushoverURL, pushoverToken, pushoverUserkey)
		notifiersCollection = append(notifiersCollection, notifier)
		log.Debug("enabled pushover notifier")
	}

	smtpArgs := notifiers.ArgsSmtpNotifier{
		To:       envFileContents[common.EnvSMTPTo].Value,
		SmtpPort: 0,
		SmtpHost: envFileContents[common.EnvSMTPHost].Value,
		From:     envFileContents[common.EnvSMTPFrom].Value,
		Password: envFileContents[common.EnvSMTPPassword].Value,
	}
	var err error
	smtpArgs.SmtpPort, err = strconv.Atoi(envFileContents[common.EnvSMTPPort].Value)
	if err != nil {
		return nil, fmt.Errorf("%w while trying to convert the .env definition SMTP_PORT value to an int", err)
	}
	if len(smtpArgs.SmtpHost) > 0 &&
		smtpArgs.SmtpPort > 0 &&
		len(smtpArgs.To) > 0 &&
		len(smtpArgs.From) > 0 &&
		len(smtpArgs.Password) > 0 {

		notifier := notifiers.NewSmtpNotifier(smtpArgs)
		notifiersCollection = append(notifiersCollection, notifier)
		log.Debug("enabled SMTP (email) notifier")
	}

	telegramBotToken := envFileContents[common.EnvTelegramBotToken]
	telegramChatId := envFileContents[common.EnvTelegramChatId]
	if len(telegramBotToken.Value) > 0 && len(telegramChatId.Value) > 0 {
		notifier := notifiers.NewTelegramNotifier(cfg.Alarms.TelegramURL, telegramBotToken.Value, telegramChatId.Value)
		notifiersCollection = append(notifiersCollection, notifier)
		log.Debug("enabled telegram notifier")
	}

	return notifiersCollection, nil
}

// GetStore returns the storage component
func (ch *componentsHandler) GetStore() api.Storage {
	return ch.store
}

// GetServer returns the server component
func (ch *componentsHandler) GetServer() Server {
	return ch.server
}

// Start starts the inner components
func (ch *componentsHandler) Start() {
	ch.server.Start()

	if !check.IfNil(ch.alarmService) {
		ch.alarmService.Start()
	}

	if !check.IfNil(ch.pollingHandlerTrigger) {
		_ = ch.pollingHandlerTrigger.StartProcessingLoop()
	}

	if !check.IfNil(ch.statusHandler) {
		ch.statusHandler.NotifyAppStart()
	}
}

// Close closes the inner components
func (ch *componentsHandler) Close() {
	_ = ch.server.Close()
	_ = ch.store.Close()

	if !check.IfNil(ch.alarmService) {
		_ = ch.alarmService.Close()
	}
	if !check.IfNil(ch.pollingHandlerTrigger) {
		_ = ch.pollingHandlerTrigger.Close()
	}

	if !check.IfNil(ch.statusHandler) {
		ch.statusHandler.SendCloseMessage()
	}
}

func parseWeekday(dayOfWeek string) (time.Weekday, error) {
	dayOfWeek = strings.ToLower(dayOfWeek)
	switch dayOfWeek {
	case "every day":
		return common.EveryWeekDay, nil
	case "monday":
		return time.Monday, nil
	case "tuesday":
		return time.Tuesday, nil
	case "wednesday":
		return time.Wednesday, nil
	case "thursday":
		return time.Thursday, nil
	case "friday":
		return time.Friday, nil
	case "saturday":
		return time.Saturday, nil
	case "sunday":
		return time.Sunday, nil
	}

	return unknownWeekDay, fmt.Errorf("unknown day of week %s", dayOfWeek)
}
