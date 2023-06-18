package bgg

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/fzerorubigd/collection-filter/internal/cache"
	"github.com/fzerorubigd/gobgg"
)

const (
	maxNumberAPiCall = 50
	cacheTime        = time.Hour * 24 * 30
)

type BGGAPI interface {
	GetThings(context.Context, ...int64) ([]gobgg.ThingResult, error)
	GetCollection(context.Context, string, bool) ([]gobgg.ThingResult, error)
}

type bggAPICached struct {
	client *gobgg.BGG
	cache  cache.Interface
}

func boardGameCacheKey(id int64) string {
	return fmt.Sprintf("boardgame_%d", id)
}

func userCacheKey(user string) string {
	return fmt.Sprintf("user_%s", user)
}

func (bca *bggAPICached) getCached(ids ...int64) map[int64]gobgg.ThingResult {
	result := make(map[int64]gobgg.ThingResult)
	for _, id := range ids {
		text := boardGameCacheKey(id)
		b, err := bca.cache.Get(text)
		if err != nil {
			continue
		}

		var thing gobgg.ThingResult
		err = json.Unmarshal(b, &thing)
		if err != nil {
			continue
		}
		result[id] = thing
	}

	return result
}

func (bca *bggAPICached) getUserCached(username string, force bool) ([]gobgg.CollectionItem, error) {
	if force {
		return nil, errors.New("force requested")
	}

	var result []gobgg.CollectionItem
	b, err := bca.cache.Get(userCacheKey(username))
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (bca *bggAPICached) getThingsMap(ctx context.Context, app map[int64]gobgg.ThingResult, req ...int64) (map[int64]gobgg.ThingResult, error) {
	if len(req) > 0 {
		things, err := bca.client.GetThings(ctx, gobgg.GetThingIDs(req...))
		if err != nil {
			return nil, err
		}
		for i := range things {
			app[things[i].ID] = things[i]
			_ = bca.cache.SetAny(boardGameCacheKey(things[i].ID), things[i], cacheTime)
		}
	}

	return app, nil
}

func (bca *bggAPICached) GetThings(ctx context.Context, ids ...int64) ([]gobgg.ThingResult, error) {
	cached := bca.getCached(ids...)
	result := make([]gobgg.ThingResult, 0, len(ids))
	req := make([]int64, 0, len(ids))
	for i := range ids {
		if _, ok := cached[ids[i]]; ok {
			continue
		}
		req = append(req, ids[i])
	}

	var err error
	for len(req) > 0 {
		if len(req) > maxNumberAPiCall {
			cached, err = bca.getThingsMap(ctx, cached, req[:maxNumberAPiCall]...)
			if err != nil {
				return nil, err
			}
			req = req[maxNumberAPiCall:]
			continue
		}
		cached, err = bca.getThingsMap(ctx, cached, req...)
		if err != nil {
			return nil, err
		}
		// we are done :/
		break
	}

	for i := range ids {
		if thing, ok := cached[ids[i]]; ok {
			result = append(result, thing)
		}
	}

	return result, nil
}

// TODO : add support for other than owned items
func (bca *bggAPICached) GetCollection(ctx context.Context, username string, force bool) ([]gobgg.ThingResult, error) {
	cached, err := bca.getUserCached(username, force)
	if err == nil {
		ids := make([]int64, len(cached))
		for i := range cached {
			ids[i] = cached[i].ID
		}

		return bca.GetThings(ctx, ids...)
	}

	col, err := bca.client.GetCollection(ctx, username, gobgg.SetCollectionTypes(gobgg.CollectionTypeOwn))
	if err != nil {
		return nil, err
	}

	_ = bca.cache.SetAny(userCacheKey(username), col, cacheTime)
	ids := make([]int64, len(col))
	for i := range col {
		ids[i] = col[i].ID
	}

	return bca.GetThings(ctx, ids...)
}

func NewCachedBGGAPI(cache cache.Interface, opt ...gobgg.OptionSetter) (BGGAPI, error) {
	return &bggAPICached{
		client: gobgg.NewBGGClient(opt...),
		cache:  cache,
	}, nil
}
