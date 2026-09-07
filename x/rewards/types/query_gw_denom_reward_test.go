package types

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

// Denom reward staking denoms are typically factory or IBC denoms, which contain "/".
// grpc-gateway v1 splits the URL path on "/" before matching, so such a denom can never be a
// path segment; the REST routes therefore take it as the `denom` query parameter.
const (
	gwFactoryDenom = "factory/bze1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq/token"
	gwIbcDenom     = "ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2"
)

// recordingQueryClient captures the request each denom reward handler forwards to the
// gRPC client so the test can assert what the gateway decoded from the URL.
type recordingQueryClient struct {
	QueryClient

	denomReward *QueryDenomRewardRequest
	prizes      *QueryDenomRewardPrizesRequest
	schedules   *QueryDenomRewardSchedulesRequest
	participant *QueryDenomRewardParticipantRequest
}

func (c *recordingQueryClient) DenomReward(_ context.Context, in *QueryDenomRewardRequest, _ ...grpc.CallOption) (*QueryDenomRewardResponse, error) {
	c.denomReward = in
	return &QueryDenomRewardResponse{}, nil
}

func (c *recordingQueryClient) DenomRewardPrizes(_ context.Context, in *QueryDenomRewardPrizesRequest, _ ...grpc.CallOption) (*QueryDenomRewardPrizesResponse, error) {
	c.prizes = in
	return &QueryDenomRewardPrizesResponse{}, nil
}

func (c *recordingQueryClient) DenomRewardSchedules(_ context.Context, in *QueryDenomRewardSchedulesRequest, _ ...grpc.CallOption) (*QueryDenomRewardSchedulesResponse, error) {
	c.schedules = in
	return &QueryDenomRewardSchedulesResponse{}, nil
}

func (c *recordingQueryClient) DenomRewardParticipant(_ context.Context, in *QueryDenomRewardParticipantRequest, _ ...grpc.CallOption) (*QueryDenomRewardParticipantResponse, error) {
	c.participant = in
	return &QueryDenomRewardParticipantResponse{}, nil
}

func newDenomRewardGateway(t *testing.T) (*runtime.ServeMux, *recordingQueryClient) {
	t.Helper()
	client := &recordingQueryClient{}
	mux := runtime.NewServeMux()
	require.NoError(t, RegisterQueryHandlerClient(context.Background(), mux, client))
	return mux, client
}

func gwGet(mux *runtime.ServeMux, path string, query url.Values) *httptest.ResponseRecorder {
	target := path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, target, nil))
	return w
}

func TestDenomRewardGateway_DenomAsQueryParam(t *testing.T) {
	for _, denom := range []string{gwFactoryDenom, gwIbcDenom} {
		t.Run(denom, func(t *testing.T) {
			mux, client := newDenomRewardGateway(t)
			q := url.Values{"denom": {denom}}

			w := gwGet(mux, "/bze/rewards/denom_reward", q)
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			require.NotNil(t, client.denomReward)
			require.Equal(t, denom, client.denomReward.Denom)

			w = gwGet(mux, "/bze/rewards/denom_reward_prizes", q)
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			require.NotNil(t, client.prizes)
			require.Equal(t, denom, client.prizes.Denom)

			w = gwGet(mux, "/bze/rewards/denom_reward_schedules", q)
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			require.NotNil(t, client.schedules)
			require.Equal(t, denom, client.schedules.Denom)

			w = gwGet(mux, "/bze/rewards/denom_reward_participant/"+testAddr, q)
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			require.NotNil(t, client.participant)
			require.Equal(t, testAddr, client.participant.Address)
			require.Equal(t, denom, client.participant.Denom)
		})
	}
}

// The previous path-segment form must not be routed anymore: with a slashed denom it never
// matched (404), and keeping it would silently leave the bug in place for plain denoms.
func TestDenomRewardGateway_DenomInPathNotRouted(t *testing.T) {
	mux, client := newDenomRewardGateway(t)

	for _, path := range []string{
		"/bze/rewards/denom_reward/ubze",
		"/bze/rewards/denom_reward/" + gwFactoryDenom,
		"/bze/rewards/denom_reward_prizes/ubze",
		"/bze/rewards/denom_reward_schedules/ubze",
		"/bze/rewards/denom_reward_participant/" + testAddr + "/ubze",
	} {
		w := gwGet(mux, path, nil)
		require.Equal(t, http.StatusNotFound, w.Code, path)
	}
	require.Nil(t, client.denomReward)
	require.Nil(t, client.prizes)
	require.Nil(t, client.schedules)
	require.Nil(t, client.participant)
}
