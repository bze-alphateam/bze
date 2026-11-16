package cli_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"cosmossdk.io/client/v2/autocli/flag"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	daov1 "github.com/bze-alphateam/bze/api/bze/dao"
	"github.com/bze-alphateam/bze/x/dao/client/cli"
)

func newConfigValue() flag.Value {
	builder := &flag.Builder{TypeResolver: protoregistry.GlobalTypes, FileResolver: protoregistry.GlobalFiles}
	ctx := context.Background()

	return cli.DecJSONMessageType{Name: "bze.dao.DaoConfig"}.NewValue(&ctx, builder)
}

func configOf(t *testing.T, v flag.Value) *daov1.DaoConfig {
	t.Helper()
	got, err := v.Get(protoreflect.Value{})
	require.NoError(t, err)
	config, ok := got.Message().Interface().(*daov1.DaoConfig)
	require.True(t, ok)

	return config
}

// Ratios are given as decimals and sent in LegacyDec's 18-decimal integer
// form; the other fields are parsed as autocli parses JSON.
func TestDecJSONMessage_DecimalsBecomeDecInts(t *testing.T) {
	v := newConfigValue()
	require.NoError(t, v.Set(`{"min_quorum":"0.5","threshold":"1","voting_period":"259200s","allow_expedite":true,`+
		`"expedited_period":"86400s","expedite_quorum":"0.600000000000000000","expedite_threshold":"0.66",`+
		`"proposal_cost":{"denom":"ubze","amount":"5"}}`))

	config := configOf(t, v)
	require.Equal(t, "500000000000000000", config.MinQuorum)
	require.Equal(t, "1000000000000000000", config.Threshold)
	require.Equal(t, "600000000000000000", config.ExpediteQuorum)
	require.Equal(t, "660000000000000000", config.ExpediteThreshold)
	require.Equal(t, int64(259200), config.VotingPeriod.Seconds)
	require.True(t, config.AllowExpedite)
	require.Equal(t, "5", config.ProposalCost.Amount)
	require.Equal(t, "bze.dao.DaoConfig (json)", v.Type())
}

// An unset ratio stays unset, for the chain's validation to refuse.
func TestDecJSONMessage_UnsetRatioStaysEmpty(t *testing.T) {
	v := newConfigValue()
	require.NoError(t, v.Set(`{"threshold":"0.5"}`))
	require.Empty(t, configOf(t, v).MinQuorum)
}

// A value that is not a decimal is refused with the field's name; invalid JSON
// is refused as autocli refuses it.
func TestDecJSONMessage_Refusals(t *testing.T) {
	v := newConfigValue()
	require.ErrorContains(t, v.Set(`{"min_quorum":"half"}`), `min_quorum: "half" is not a decimal`)
	require.Error(t, v.Set(`{"min_quorum":`))
	require.Error(t, v.Set(`{"unknown_field":"1"}`))
	require.ErrorContains(t, newConfigValue().Set(filepath.Join(t.TempDir(), "missing.json")), "no such file")
}

// A .json argument is read from the file.
func TestDecJSONMessage_ReadsJSONFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"min_quorum":"0.25"}`), 0o600))

	v := newConfigValue()
	require.NoError(t, v.Set(path))
	require.Equal(t, "250000000000000000", configOf(t, v).MinQuorum)
	require.Contains(t, v.String(), "250000000000000000")
}

// An unknown message name is refused when the value is set.
func TestDecJSONMessage_UnknownMessage(t *testing.T) {
	builder := &flag.Builder{TypeResolver: protoregistry.GlobalTypes, FileResolver: protoregistry.GlobalFiles}
	ctx := context.Background()
	v := cli.DecJSONMessageType{Name: "bze.dao.Nope"}.NewValue(&ctx, builder)
	require.ErrorContains(t, v.Set(`{}`), "bze.dao.Nope not found")
	require.Empty(t, v.String())
}
