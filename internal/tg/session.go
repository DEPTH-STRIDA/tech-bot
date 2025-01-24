package tg

import (
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/patrickmn/go-cache"
)

const (
	defaultExpiration = 24 * time.Hour
	cleanupInterval   = 30 * time.Minute
)

type CachedUser struct {
	MailingType        MailingType
	MailingMessagetext string
	CohortsIndex       int
	CohortsName        string
	Entities           []tgbotapi.MessageEntity
}

type SessionsCache struct {
	cache *cache.Cache
}

func NewSessionsCache() *SessionsCache {
	return &SessionsCache{
		cache: cache.New(defaultExpiration, cleanupInterval),
	}
}

func (sc *SessionsCache) GetAll() map[string]CachedUser {
	items := sc.cache.Items()
	result := make(map[string]CachedUser)

	for k, v := range items {
		if user, ok := v.Object.(CachedUser); ok {
			result[k] = user
		}
	}

	return result
}

func keyToString(key int64) string {
	return fmt.Sprintf("%d", key)
}

func (sc *SessionsCache) Set(key int64, user CachedUser) {
	sc.cache.Set(keyToString(key), user, defaultExpiration)
}

// Get возвращает CachedUser из кэша. Если пользователь не найден,
// создает новую пустую структуру, сохраняет её в кэше и возвращает
func (sc *SessionsCache) Get(key int64) CachedUser {
	value, exists := sc.cache.Get(keyToString(key))
	if !exists {
		// Создаем пустую структуру
		newUser := CachedUser{}
		// Сохраняем её в кэше
		sc.Set(key, newUser)
		return newUser
	}

	user, ok := value.(CachedUser)
	if !ok {
		// В случае ошибки приведения типа также возвращаем новую структуру
		newUser := CachedUser{}
		sc.Set(key, newUser)
		return newUser
	}

	return user
}

func (sc *SessionsCache) Delete(key int64) {
	sc.cache.Delete(keyToString(key))
}
