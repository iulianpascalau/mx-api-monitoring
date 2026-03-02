package common

import "time"

// ExecutorName defines the alarm naming used in the notification messages
const ExecutorName = "API monitoring app"

// EveryWeekDay is the constant that encodes each week day option
const EveryWeekDay = time.Weekday(-1)

const (
	EnvServiceKey       = "SERVICE_KEY"
	EnvAuthUser         = "AUTH_USER"
	EnvAuthPassword     = "AUTH_PASSWORD"
	EnvPushoverToken    = "PUSHOVER_TOKEN"
	EnvPushoverUserKey  = "PUSHOVER_USERKEY"
	EnvSMTPTo           = "SMTP_TO"
	EnvSMTPFrom         = "SMTP_FROM"
	EnvSMTPPassword     = "SMTP_PASSWORD"
	EnvSMTPPort         = "SMTP_PORT"
	EnvSMTPHost         = "SMTP_HOST"
	EnvTelegramBotToken = "TELEGRAM_BOT_TOKEN"
	EnvTelegramChatId   = "TELEGRAM_CHAT_ID"
)
