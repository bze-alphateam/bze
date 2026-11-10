package v830

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"

	corestore "cosmossdk.io/core/store"
	storetypes "cosmossdk.io/store/types"
	cmtbytes "github.com/cometbft/cometbft/libs/bytes"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	"google.golang.org/protobuf/encoding/protowire"
)

// TransferKeeper is the part of the ibc-go v10 transfer keeper the handler uses.
type TransferKeeper interface {
	SetDenom(ctx sdk.Context, denom transfertypes.Denom)
}

// Field numbers of ibc.applications.transfer.v1.DenomTrace, the record ibc-go v8
// stored under transfertypes.DenomTraceKey.
const (
	denomTracePathField protowire.Number = 1
	denomTraceBaseField protowire.Number = 2
)

// denomTrace is a stored ibc-go v8 denom trace.
type denomTrace struct {
	key  []byte // the store key under DenomTraceKey: sha256 of the full path
	path string
	base string
}

// fullPath is the path the voucher's ibc/{hash} denom is computed from: path + "/" + base,
// or the base alone for an empty path.
func (dt denomTrace) fullPath() string {
	if dt.path == "" {
		return dt.base
	}

	return dt.path + "/" + dt.base
}

// ibcDenom is the bank denom ibc-go v8 gave the voucher of this trace.
func (dt denomTrace) ibcDenom() string {
	if dt.path == "" {
		return dt.base
	}
	hash := sha256.Sum256([]byte(dt.fullPath()))

	return fmt.Sprintf("%s/%s", transfertypes.DenomPrefix, cmtbytes.HexBytes(hash[:]))
}

// sanitizeDenomTraces prepares the ibc-go v8 denom traces for the transfer store migration that
// RunMigrations runs (MigrateDenomTraceToDenom). That migration re-parses each trace's full path with
// the ibc-go v10 rules, which differ from the v8 rules the trace was stored with, and accepts the
// result only when it validates and keeps the voucher's bank denom.
//
// Traces the migration accepts are left for it. Every other trace is converted here to the v10 Denom
// whose path is the stored full path, so the voucher denom ibc/{sha256(full path)} and every balance
// held in it stay the same, and its trace record is removed so the migration does not see it again.
// The conversion keeps as many of the stored port/channel hops as validate, folding the rest into the
// base denom. A trace with no such Denom (or one that cannot be decoded) is deleted and logged.
//
// The store is read in key order and written after the iterator is closed. It runs once, over every
// trace the chain has received, before RunMigrations.
func sanitizeDenomTraces(ctx sdk.Context, transferStore corestore.KVStoreService, transfer TransferKeeper) {
	store := runtime.KVStoreAdapter(transferStore.OpenKVStore(ctx))
	traceKey := func(hash []byte) []byte { return append(append([]byte{}, transfertypes.DenomTraceKey...), hash...) }

	var (
		traces      []denomTrace
		undecodable [][]byte
	)
	iterator := storetypes.KVStorePrefixIterator(store, transfertypes.DenomTraceKey)
	for ; iterator.Valid(); iterator.Next() {
		hash := append([]byte{}, iterator.Key()[len(transfertypes.DenomTraceKey):]...)
		path, base, err := decodeDenomTrace(iterator.Value())
		if err != nil {
			undecodable = append(undecodable, hash)
			continue
		}
		traces = append(traces, denomTrace{key: hash, path: path, base: base})
	}
	if err := iterator.Close(); err != nil {
		panic(err)
	}

	kept, converted, deleted := 0, 0, 0
	for _, hash := range undecodable {
		store.Delete(traceKey(hash))
		deleted++
		ctx.Logger().Info("deleted undecodable denom trace", "hash", cmtbytes.HexBytes(hash).String())
	}

	for _, dt := range traces {
		if migratesAsIs(dt) {
			kept++
			continue
		}

		denom, ok := convertDenomTrace(dt)
		if ok {
			transfer.SetDenom(ctx, denom)
			converted++
			ctx.Logger().Info("converted denom trace", "denom", dt.ibcDenom(), "trace", denom.Trace, "base", denom.Base)
		} else {
			deleted++
			ctx.Logger().Info("deleted denom trace without a valid denom", "denom", dt.ibcDenom(), "path", dt.path, "base", dt.base)
		}
		store.Delete(traceKey(dt.key))
	}

	ctx.Logger().Info("sanitized ibc denom traces", "kept", kept, "converted", converted, "deleted", deleted)
}

// migratesAsIs reports whether the transfer store migration accepts the trace: the v10 parse of its
// full path validates and keeps the voucher denom.
func migratesAsIs(dt denomTrace) bool {
	denom := transfertypes.ExtractDenomFromPath(dt.fullPath())

	return denom.Validate() == nil && denom.IBCDenom() == dt.ibcDenom()
}

// convertDenomTrace returns the valid Denom with the trace's full path and store key and the most hops,
// taken from the stored path. A trace with an empty path is a native denom and has no conversion.
func convertDenomTrace(dt denomTrace) (transfertypes.Denom, bool) {
	// Belt and braces: a trace with an empty path already migrates as is unless its base is unparsable.
	if dt.path == "" {
		return transfertypes.Denom{}, false
	}

	segments := strings.Split(dt.path, "/")
	if len(segments)%2 != 0 {
		return transfertypes.Denom{}, false
	}
	hops := make([]transfertypes.Hop, 0, len(segments)/2)
	for i := 0; i < len(segments); i += 2 {
		hops = append(hops, transfertypes.NewHop(segments[i], segments[i+1]))
	}

	for n := len(hops); n > 0; n-- {
		folded := make([]string, 0, len(hops)-n+1)
		for _, hop := range hops[n:] {
			folded = append(folded, hop.String())
		}
		folded = append(folded, dt.base)

		denom := transfertypes.NewDenom(strings.Join(folded, "/"), hops[:n]...)
		// The path and IBC denom checks hold by construction; they stay as defence in depth.
		if denom.Validate() == nil && denom.Path() == dt.fullPath() && denom.IBCDenom() == dt.ibcDenom() &&
			bytes.Equal(denom.Hash(), dt.key) {
			return denom, true
		}
	}

	return transfertypes.Denom{}, false
}

// decodeDenomTrace reads the path and base denom of a stored ibc-go v8 DenomTrace.
func decodeDenomTrace(bz []byte) (path, base string, err error) {
	for len(bz) > 0 {
		num, typ, n := protowire.ConsumeTag(bz)
		if n < 0 {
			return "", "", protowire.ParseError(n)
		}
		bz = bz[n:]

		if typ == protowire.BytesType && (num == denomTracePathField || num == denomTraceBaseField) {
			v, m := protowire.ConsumeString(bz)
			if m < 0 {
				return "", "", protowire.ParseError(m)
			}
			if num == denomTracePathField {
				path = v
			} else {
				base = v
			}
			bz = bz[m:]
			continue
		}

		m := protowire.ConsumeFieldValue(num, typ, bz)
		if m < 0 {
			return "", "", protowire.ParseError(m)
		}
		bz = bz[m:]
	}
	// Belt and braces: an empty record has no denom to convert.
	if path == "" && base == "" {
		return "", "", errors.New("empty denom trace")
	}

	return path, base, nil
}
