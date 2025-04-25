package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"strings"
)

const (
	// ModuleName defines the module name
	ModuleName = "cointrunk"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_cointrunk"

	// PublisherKeyPrefix is the prefix to retrieve all Publisher
	PublisherKeyPrefix = "Publisher/value/"
	// ArticleKeyPrefix is the prefix to retrieve all Article
	ArticleKeyPrefix             = "Article/value/"
	ArticleCounterKeyPrefix      = "Article/counter/"
	AnonArticlesCounterKeyPrefix = "Article/anon/counter/"
	// AcceptedDomainKeyPrefix is the prefix to retrieve all AcceptedDomain
	AcceptedDomainKeyPrefix = "AcceptedDomain/value/"
)

var (
	ParamsKey = []byte("p_cointrunk")
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}

// PublisherKey returns the store key to retrieve a Publisher from the index fields
func PublisherKey(
	index string,
) []byte {
	var key []byte

	indexBytes := []byte(index)
	key = append(key, indexBytes...)
	key = append(key, []byte("/")...)

	return key
}

// ArticleKey returns the store key to retrieve a Publisher from the index fields
func ArticleKey(
	index string,
) []byte {
	var key []byte

	indexBytes := []byte(index)
	key = append(key, indexBytes...)
	key = append(key, []byte("/")...)

	return key
}

// AcceptedDomainKey returns the store key to retrieve an AcceptedDomain from the index fields
func AcceptedDomainKey(
	index string,
) []byte {
	index = strings.TrimPrefix(index, "www.")
	var key []byte

	indexBytes := []byte(index)
	key = append(key, indexBytes...)
	key = append(key, []byte("/")...)

	return key
}

func GenerateMonthlyPaidArticleCounterPrefix(ctx sdk.Context) (prefix string) {
	return AnonArticlesCounterKeyPrefix + ctx.BlockHeader().Time.Format("200601")
}
