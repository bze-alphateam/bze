package types

import (
	"encoding/binary"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ binary.ByteOrder

const (
	// ArticleKeyPrefix is the prefix to retrieve all Article
	ArticleKeyPrefix = "Article/value/"
)

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

func GenerateArticlePrefix(ctx sdk.Context) (prefix string) {
	return ArticleKeyPrefix + ctx.BlockHeader().Time.Format("200601021506")
}

func GenerateArticleCountPrefix(ctx sdk.Context) (prefix string) {
	return ctx.BlockHeader().Time.Format("200601")
}
