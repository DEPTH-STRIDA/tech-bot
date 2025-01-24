package config

import (
	"fmt"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	WebConfig
	TelegramConfig
	GoogleSheetConfig
	CacheConfig
	ClickHouseConfig
	DataBaseConfig
	EasyCodeApiConfig
	NotificationConfig
	LoggerConfig
}

type WebConfig struct {
	APPIP             string `envconfig:"APP_IP" default:"localhost"`             // IP адрес приложения
	APPPORT           string `envconfig:"APP_PORT" default:"8080"`                // Порт приложения
	FormIDLength      int    `envconfig:"APP_FORM_ID_LENGTH" default:"8"`         // Длина ID заявки на замену-перенос
	NumberRepetitions int    `envconfig:"APP_NUMBER_OF_REPETITIONS" default:"15"` // Количество повторов запроса на замену-перенос
	RepeatPause       int    `envconfig:"APP_REPEAT_PAUSE" default:"15"`          // Пауза между повторами запроса на замену-перенос
	MentoringManager  string `envconfig:"APP_MENTORING_MANAGER" default:""`       // Имя менеджера по наставничеству
	IsTestMode        bool   `envconfig:"APP_IS_TEST_MODE" default:"false"`       // является ли запуск приложения
}

type TelegramConfig struct {
	Token                      string `envconfig:"TELEGRAM_TOKEN" required:"true"`                                  // Токен бота
	NotificationChatId         int64  `envconfig:"TELEGRAM_NOTIFICATION_CHAT_ID" required:"true"`                   // ID тг чата, куда будут приходить уведомления о пропусках, ошибка и т.г.
	ReplaceChatID              int64  `envconfig:"TELEGRAM_REPLACE_CHAT_ID" required:"true"`                        // ID чата для замен. В этот чат отправляются все замены-переносы
	EmergencyChatID            int64  `envconfig:"TELEGRAM_EMERGENCY_CHAT_ID" required:"true"`                      // ID чата для экстренных замен
	WebAppUrl                  string `envconfig:"TELEGRAM_WEB_APP_URL" required:"true"`                            // URL веб приложения
	StartStickerID             string `envconfig:"TELEGRAM_START_STICKER_ID" required:"true"`                       // ID стикера для стартового сообщения
	StartMsg                   string `envconfig:"TELEGRAM_START_MESSAGE" required:"true"`                          // Стартовое сообщение
	MenuMsg                    string `envconfig:"TELEGRAM_MENU_MESSAGE" required:"true"`                           // Сообщение меню
	ButtonMenuMsg              string `envconfig:"TELEGRAM_BUTTON_MENU_MESSAGE"`                                    // Сообщение кнопки меню
	PinUpMsg                   string `envconfig:"TELEGRAM_PINUP_MESSAGE" required:"true"`                          // Сообщение, которое будет закреплено в чате                                           // Список ID чатов админов
	RequestUpdatePause         int    `envconfig:"TELEGRAM_REQUEST_UPDATE_PAUSE_MIILISECOND" default:"1000"`        // Пауза между обработкой обновлений. Например, отправки сообщений
	RequestCallBackUpdatePause int    `envconfig:"TELEGRAM_REQUEST_CALLBACK_UPDATE_PAUSE_MIILISECOND default:"500"` // Пауза между запросами на обновление callback
	MsgBufferSize              int    `envconfig:"TELEGRAM_MESSAGE_BUFFER_SIZE" default:"100"`                      // Размер буфера для сообщений
	CallBackBufferSize         int    `envconfig:"TELEGRAM_CALLBACK_BUFFER_SIZE" default:"100"`                     // Размер буфера для callback
	BotTgChat                  int64  `envconfig:"TELEGRAM_BOT_TG_CHAT" required:"true"`                            // ID чата основного чата, где сообщения распределяются по темам.
	ErrorTopicID               int    `envconfig:"TELEGRAM_ERROR_TOPIC_ID" required:"true"`                         // ID сообщения темы для ошибок
	AbsenteeismTopicID         int    `envconfig:"TELEGRAM_ABSENTEEISM_TOPIC_ID" required:"true"`                   // ID сообщения темы для отсутствий
	StatTopicId                int    `envconfig:"TELEGRAM_STAT_TOPIC_ID" required:"true"`                          // ID сообщения темы для статистики
	BirthTopicId               int    `envconfig:"TELEGRAM_BIRTH_TOPIC_ID" required:"true"`                         // ID сообщения темы для ДР
}

type GoogleSheetConfig struct {
	CredentialsFile    string   `envconfig:"SHEET_CREDENTIALS_FILE" required:"true"`
	RequiredFreeFields []string `envconfig:"SHEET_REQUIRED_FREE_FIELDS" required:"true"` // Столбцы, которые должны быть свободны в таблице, чтобы строка считалась свободной и была доступна для внесения заявки на замену-перенос
	ReplaceTableID     string   `envconfig:"SHEET_REPLACE_TABLE_ID" required:"true"`     // ID таблицы для замен
	ReplaceListName    string   `envconfig:"SHEET_REPLACE_LIST_NAME" required:"true"`    // Название листа для замен

	SelectDataTableID  string `envconfig:"SHEET_SELECT_DATA_TABLE_ID" required:"true"`  // ID таблицы, где хранятся данные для выпадающих список мини приложения
	SelectDataListName string `envconfig:"SHEET_SELECT_DATA_LIST_NAME" required:"true"` // Название листа для, где хранятся данные для выпадающих список мини приложения

	BirthDayDataTableID  string `envconfig:"SHEET_BIRTH_DATA_TABLE_ID" required:"true"`  // ID таблицы, где хранятся данные преподавателей. В том числе и ДР
	BirthDayDataListName string `envconfig:"SHEET_BIRTH_DATA_LIST_NAME" required:"true"` // Название листа для, где хранятся данные преподавателей. В том числе и ДР

	AdminDataListName         string   `envconfig:"SHEET_ADMINS_LIST_NAME" required:"true"`              // Название листа для, где хранятся данные для админов
	SelectDataColumNames      []string `envconfig:"SHEET_SELECT_DATA_COLUMN_NAMES" required:"true"`      // Столбцы, где хранятся данные для выпадающих список мини приложения
	RequestUpdatePause        int      `envconfig:"SHEET_REQUEST_UPDATE_PAUSE" default:"15"`             // Пауза между выполнением запросов
	SelectDataUpdatePauseHour int      `envconfig:"SHEET_SELECT_DATA_UPDATE_PAUSE_HOUR" required:"true"` // Пауза между обновлением данных выпадающих списков мини приложения
	BufferSize                int      `envconfig:"SHEET_BUFFER_SIZE default:"100""`                     // Размер буфера для отложенных событий. Можно создать и не буферезированный, но стоит задать некоторое число
	EmergencyCellName         string   `envconfig:"SHEET_EMERGENCY_CELL_NAME" required:"true"`           // Имя ячейки, где хранится bool статус замены. Срочная ли
	TeachersColectRepet       int      `envconfig:"SHEET_TEACHERS_COLECT_REPET" default:"15"`            // Количество раз, которое нужно собрать данные о преподавателях
	UsersListName             string   `envconfig:"SHEET_USERS_LIST_NAME" required:"true"`               // Название листа, где хранятся данные преподавателей.
}

type CacheConfig struct {
	ReplaceFormLiveTime   int `envconfig:"CACHE_REPLACE_LIVE_TIME" default:"12"`  // Время жизни данных в кэше для заявок на замену-перенос
	EmergencyFormLiveTime int `envconfig:"CACHE_EMERGENCY_LIVE_TIME" default:"6"` // Время жизни данных в кэше для экстренных ззамен
}

type ClickHouseConfig struct {
	DBHost                   string  `envconfig:"CLICKHOUSE_HOST" required:"true"` // IP адресс для подключения к БД
	DBName                   string  `envconfig:"CLICKHOUSE_DB_NAME" required:"true"`
	DBUser                   string  `envconfig:"CLICKHOUSE_USER" required:"true"`
	DBPort                   string  `envconfig:"CLICKHOUSE_PORT" required:"true"`
	DBPass                   string  `envconfig:"CLICKHOUSE_PASS" required:"true"`
	DBNumberRepetitions      int     `envconfig:"CLICKHOUSE_DB_NUMBER_OF_REPETITIONS" default:"15"`
	DBFormTableName          string  `envconfig:"CLICKHOUSE_FORM_TABLE_NAME" required:"true"`
	CHPauseAfterSQLExecute   float32 `envconfig:"CLICKHOUSE_CH_PAUSE_AFTER_SQL_EXECUTE" default:"3"`
	PauseAfterFailConnection int     `envconfig:"CLICKHOUSE_PAUSE_AFTER_FAIL_CONNECTION_SEC" default:"3"`
}

type DataBaseConfig struct {
	Host     string `envconfig:"DBHOST" required:"true"` // IP адресс для подключение к БД
	Port     string `envconfig:"DBPORT" default:""`      // Port для подключение к БД
	DBName   string `envconfig:"DBNAME" required:"true"` // Имя базы данных
	UserName string `envconfig:"DBUSER" required:"true"` // Имя пользователя
	Password string `envconfig:"DBPASS" required:"true"` // Пароль пользователя
	SSLMode  string `envconfig:"DBSSLMODE" default:"disable"`
}

type EasyCodeApiConfig struct {
	AccessToken     string `envconfig:"EASYCODEAPI_ACCESS_TOKEN" required:"true"`   //  Токен для доступа к API EasyCode
	MemberAPIURL    string `envconfig:"EASYCODEAPI_MEMBER_API_URL" required:"true"` // URL для получения списка участников группы
	LessonsAPIURL   string `envconfig:"EASYCODEAPI_LESSONS_API_URL" required:"true` // URL для получения списка уроков за дату
	ApiRequestPause int64  `envconfig:"EASYCODEAPI_API_REQUEST_PAUSE" default:"2"`  // Пауза между запросами к API EasyCode
	ApiBufferSize   int    `envconfig:"EASYCODEAPI_API_BUFFER_SIZE" default:"500"`  // Размер буфера для запросов к API EasyCode
}

type NotificationConfig struct {
	BeforeLessonNotificationTime int `envconfig:"NOTIFICATION_BEFORE_LESSON_NOTIFICATION_TIME" required:"true"` // За сколько минут до урока отправлять уведомления
	MorningUpdateTime            int `envconfig:"NOTIFICATION_MORNING_UPDATE_TIME" default:"7"`                 // Время утром, когда будет отправляться уведомление об уроках за день
	AfterSendPause               int `envconfig:"NOTIFICATION_AFTER_SEND_PAUSE" default:"60"`                   // Пауза после отправки уведомления
	AfterCheckpause              int `envconfig:"NOTIFICATION_AFTER_CHECK_PAUSES" default:"60"`                 // Пауза после проверки наличия уроков
	DelayTime                    int `envconfig:"NOTIFICATION_DELAY_TIME" default:"15"`
}

// internal/model/config.go
type LoggerConfig struct {
	LogDir      string `envconfig:"LOG_DIR" default:"./log/tech_bot"`
	MaxFileSize int64  `envconfig:"LOG_MAX_FILE_SIZE" default:"10485760"` // 10MB в байтах
	TimeFormat  string `envconfig:"LOG_TIME_FORMAT" default:"2006-01-02_15-04-05"`
	FilePattern string `envconfig:"LOG_FILE_PATTERN" default:"tech_bot_%s.log"`
}

var File *Config

func init() {
	// Загрузка файла .env
	if err := godotenv.Load("../../config/tech_bot/.env"); err != nil {
		panic(err)
	}

	File = &Config{}
	err := envconfig.Process("", File)
	if err != nil {
		panic(err)
	}
	fmt.Println("Загруженые параметры: \n", File)
}
