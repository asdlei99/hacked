package media

import (
	"github.com/inkyblackness/hacked/ss1/content/text"
	"github.com/inkyblackness/hacked/ss1/resource"
	"github.com/inkyblackness/hacked/ss1/world/ids"
)

// TODO: type for byte arrays? -> ss1/resource

// TextBlockGetter provides raw data of blocks.
type TextBlockGetter interface {
	ModifiedBlock(lang resource.Language, id resource.ID, index int) []byte
	ModifiedBlocks(lang resource.Language, id resource.ID) [][]byte
}

// TextService is modding text.
type GetTextService struct {
	lineCache *text.Cache
	pageCache *text.Cache
	getter    TextBlockGetter
}

func NewGetTextService(lineCache, pageCache *text.Cache, getter TextBlockGetter) GetTextService {
	return GetTextService{
		lineCache: lineCache,
		pageCache: pageCache,
		getter:    getter,
	}
}

func (service GetTextService) Current(key resource.Key) string {
	var cache *text.Cache
	resourceInfo, existing := ids.Info(key.ID)
	if !existing || resourceInfo.List {
		cache = service.lineCache
	} else {
		cache = service.pageCache
	}
	currentValue, cacheErr := cache.Text(key)
	if cacheErr != nil {
		currentValue = ""
	}
	return currentValue
}

func (service GetTextService) Modified(key resource.Key) bool {
	var data [][]byte
	info, _ := ids.Info(key.ID)
	if info.List {
		data = [][]byte{service.getter.ModifiedBlock(key.Lang, key.ID, key.Index)}
	} else {
		data = service.getter.ModifiedBlocks(key.Lang, key.ID.Plus(key.Index))
	}
	return len(data) > 0
}