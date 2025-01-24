package cache

import (
	"sync"
)

// TelegramCacheApp кэш для ТГ админов
var TelegramCacheApp *TelegramCache = &TelegramCache{
	TelegramCacheContent: TelegramCacheContent{
		TgAdminIDS: []int64{878413772},
		Cohorts:    [][]string{{"Тестовая кагорта"}},
		TeamChats:  []int64{878413772, 878413772, 878413772, 878413772},
	},
}

// TelegramCache кэш для ТГ админов
type TelegramCache struct {
	sync.RWMutex
	TelegramCacheContent
}

// TelegramCacheContent данные ТГ админов
type TelegramCacheContent struct {
	TgAdminIDS []int64
	Cohorts    [][]string
	TeamChats  []int64
}

// TelegramCache возвращает ТГ админов
func (cache *TelegramCache) GetTgAdmins() []int64 {
	cache.RLock()
	defer cache.RUnlock()

	return cache.TgAdminIDS
}

// IsAdmin проверяет, что пользователь администратор
func (cache *TelegramCache) IsAdmin(telegramUserID int64) bool {
	cache.RLock()
	defer cache.RUnlock()

	for _, v := range cache.TgAdminIDS {
		if v == telegramUserID {
			return true
		}
	}

	return false
}

// GetCohortsNames возвращает список кагорт
func (cache *TelegramCache) GetCohortsNames() []string {
	cache.RLock()
	defer cache.RUnlock()

	// Срез названией кагорт
	name := make([]string, 0)

	// Добавляем название кагорты в срез. Название - первый элемент кагорты
	for _, v := range cache.Cohorts {
		if len(v) != 0 {
			name = append(name, v[0])
		}
	}

	return name
}

// GetCohortByName возвращает содержимое кагорты по ее названию
func (cache *TelegramCache) GetCohortByName(name string) []string {
	cache.RLock()
	defer cache.RUnlock()

	for _, v := range cache.Cohorts {
		if len(v) != 0 {
			if v[0] == name {
				return v
			}
		}
	}

	return make([]string, 0)
}

// GetTeamChats возвращает id тг командных чатов
func (cache *TelegramCache) GetTeamChats() []int64 {
	cache.RLock()
	defer cache.RUnlock()

	return cache.TeamChats
}
