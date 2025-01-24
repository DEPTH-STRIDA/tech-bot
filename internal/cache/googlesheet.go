package cache

import (
	"sync"
)

// GoogleSheetCacheApp кэш для выпадающих списков
var GoogleSheetCacheApp *GoogleSheetCache = &GoogleSheetCache{
	SelectDataContent: SelectDataContent{
		Teachers:            make([]string, 0),
		Objects:             make([]string, 0),
		ReplacementFormats:  make([]string, 0),
		TransfermentFormats: make([]string, 0),
		TeamLeaders:         make([]string, 0),
	},
}

// GoogleSheetCache кэш для выпадающих списков
type GoogleSheetCache struct {
	sync.RWMutex
	SelectDataContent
}

// SelectDataContent данные выпадающих списков
type SelectDataContent struct {
	Teachers            []string
	Objects             []string
	ReplacementFormats  []string
	TransfermentFormats []string
	TeamLeaders         []string
}

// GetSelectData возвращает данные выпадающих списков
func (cache *GoogleSheetCache) GetSelectData() SelectDataContent {
	cache.RLock()
	defer cache.RUnlock()

	return cache.SelectDataContent
}
