package types_test

import (
	"encoding/json"
	"testing"

	"cosmossdk.io/math"
	"cosmossdk.io/x/tx/signing/aminojson"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	tradebinapi "github.com/bze-alphateam/bze/api/bze/tradebin"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	"github.com/bze-alphateam/bze/x/tradebin/v2types"
)

// queryParamsJSON prints a params query response the way `bzed q tradebin
// params -o json` does (autocli: the pulsar type through the amino JSON
// encoder) and returns its "params" object.
func queryParamsJSON(t *testing.T, p v2types.Params) map[string]json.RawMessage {
	t.Helper()
	bz, err := (&types.QueryParamsResponse{Params: p}).Marshal()
	require.NoError(t, err)
	var res tradebinapi.QueryParamsResponse
	require.NoError(t, proto.Unmarshal(bz, &res))

	enc := aminojson.NewEncoder(aminojson.EncoderOptions{
		EnumAsString:       true,
		DoNotSortFields:    true,
		AminoNameAsTypeURL: true,
		MarshalMappings:    true,
	})
	out, err := enc.Marshal(&res)
	require.NoError(t, err)

	var printed struct {
		Params map[string]json.RawMessage `json:"params"`
	}
	require.NoError(t, json.Unmarshal(out, &printed))
	return printed.Params
}

// decodeUpdateParams decodes a MsgUpdateParams JSON the way a proposal file or
// a JSON tx is read (gogo jsonpb through the proto codec).
func decodeUpdateParams(t *testing.T, params map[string]json.RawMessage) v2types.Params {
	t.Helper()
	bz, err := json.Marshal(map[string]any{"authority": "gov", "params": params})
	require.NoError(t, err)
	var msg types.MsgUpdateParams
	require.NoError(t, codec.NewProtoCodec(codectypes.NewInterfaceRegistry()).UnmarshalJSON(bz, &msg))
	return msg.Params
}

// The price drift limits are printed as decimals by the params query.
func TestParamsQueryJSON_DriftLimitsAreDecimals(t *testing.T) {
	p := v2types.DefaultParams()
	p.FeeMaxPriceDrift = math.LegacyMustNewDecFromStr("0.07")
	p.ModuleMaxPriceDrift = math.LegacyMustNewDecFromStr("0.000000000000000001")

	printed := queryParamsJSON(t, p)
	require.JSONEq(t, `"0.070000000000000000"`, string(printed["feeMaxPriceDrift"]))
	require.JSONEq(t, `"0.000000000000000001"`, string(printed["moduleMaxPriceDrift"]))
}

// The params a query prints, sent back unchanged in a MsgUpdateParams, are
// the same valid params: a governance proposal can be built from the query
// output by changing only the field it is about.
func TestParamsQueryJSON_RoundTripsIntoUpdateParams(t *testing.T) {
	p := v2types.DefaultParams()
	p.FeeMaxPriceDrift = math.LegacyMustNewDecFromStr("0.25")
	p.ModuleMaxPriceDrift = math.LegacyMustNewDecFromStr("0.015")
	p.MaxFillsPerOrder = 7

	printed := queryParamsJSON(t, p)
	got := decodeUpdateParams(t, printed)
	require.NoError(t, got.Validate())
	want, err := p.Marshal()
	require.NoError(t, err)
	gotBz, err := got.Marshal()
	require.NoError(t, err)
	require.Equal(t, want, gotBz, "round trip changed the params:\nwant %+v\ngot  %+v", p, got)

	printed["maxFillsPerOrder"] = json.RawMessage(`"9"`)
	got = decodeUpdateParams(t, printed)
	require.NoError(t, got.Validate())
	require.Equal(t, uint64(9), got.MaxFillsPerOrder)
	require.True(t, p.FeeMaxPriceDrift.Equal(got.FeeMaxPriceDrift), "fee drift kept: %s", got.FeeMaxPriceDrift)
	require.True(t, p.ModuleMaxPriceDrift.Equal(got.ModuleMaxPriceDrift), "module drift kept: %s", got.ModuleMaxPriceDrift)
}
