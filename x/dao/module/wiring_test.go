package dao_test

import (
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	apidao "github.com/bze-alphateam/bze/api/bze/dao"
	"github.com/bze-alphateam/bze/x/dao/keeper"
	dao "github.com/bze-alphateam/bze/x/dao/module"
	"github.com/bze-alphateam/bze/x/dao/types"
)

// The module's services go through the real SDK plumbing: MsgUpdateParams is
// routed by the MsgServiceRouter and the Params query by the GRPCQueryRouter.
func TestRegisterServices_RoutesMsgAndQuery(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	msr := baseapp.NewMsgServiceRouter()
	msr.SetInterfaceRegistry(registry)

	k := keeper.NewKeeper(cdc, runtime.NewKVStoreService(storeKey), log.NewNopLogger(), authority, nil, nil, nil, msr)
	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	qr := baseapp.NewGRPCQueryRouter()
	qr.SetInterfaceRegistry(registry)
	mm := module.NewManager(dao.NewAppModule(cdc, &k, nil, nil))
	require.NoError(t, mm.RegisterServices(module.NewConfigurator(cdc, msr, qr)))

	msg := &types.MsgUpdateParams{Authority: authority, Params: types.DefaultParams()}
	handler := msr.Handler(msg)
	require.NotNil(t, handler)
	_, err := handler(ctx, msg)
	require.NoError(t, err)

	bad := &types.MsgUpdateParams{Authority: sdk.AccAddress("not_the_gov_account_").String(), Params: types.DefaultParams()}
	_, err = handler(ctx, bad)
	require.ErrorIs(t, err, types.ErrInvalidSigner)

	require.NotNil(t, qr.Route("/bze.dao.Query/Params"))
	require.NotNil(t, qr.Route("/bze.dao.Query/Dao"))
	require.NotNil(t, qr.Route("/bze.dao.Query/DaoByAddress"))
	require.NotNil(t, qr.Route("/bze.dao.Query/DaoController"))
	require.NotNil(t, qr.Route("/bze.dao.Query/Proposal"))
	require.NotNil(t, qr.Route("/bze.dao.Query/Ballot"))
	require.NotNil(t, qr.Route("/bze.dao.Query/VotingPower"))
	require.NotNil(t, qr.Route("/bze.dao.Query/JoinRecord"))
	require.NotNil(t, qr.Route("/bze.dao.Query/Daos"))
	require.NotNil(t, qr.Route("/bze.dao.Query/DaosByAdmin"))
	require.NotNil(t, qr.Route("/bze.dao.Query/Proposals"))
	require.NotNil(t, qr.Route("/bze.dao.Query/Ballots"))
}

// The CLI exposes `bzed q dao params` and `bzed tx dao update-params`; the
// latter is authority gated, so it is submitted as a governance proposal.
func TestAutoCLIOptions(t *testing.T) {
	opts := dao.AppModule{}.AutoCLIOptions()

	require.Equal(t, "bze.dao.Query", opts.Query.Service)
	require.Len(t, opts.Query.RpcCommandOptions, 12)
	require.Equal(t, "Params", opts.Query.RpcCommandOptions[0].RpcMethod)
	require.Equal(t, "params", opts.Query.RpcCommandOptions[0].Use)
	for i, want := range []struct {
		method, use string
		args        []string
	}{
		{"Dao", "dao [id]", []string{"id"}},
		{"DaoByAddress", "dao-by-address [address]", []string{"address"}},
		{"DaoController", "dao-controller [dao-id]", []string{"dao_id"}},
		{"Proposal", "proposal [proposal-id]", []string{"id"}},
		{"Ballot", "ballot [proposal-id] [voter]", []string{"proposal_id", "voter"}},
		{"VotingPower", "voting-power [dao-id] [address]", []string{"dao_id", "address"}},
		{"JoinRecord", "join-record [source] [address]", []string{"source", "address"}},
		{"Daos", "daos", nil},
		{"DaosByAdmin", "daos-by-admin [admin]", []string{"admin"}},
		{"Proposals", "proposals [dao-id]", []string{"dao_id"}},
		{"Ballots", "ballots [proposal-id]", []string{"proposal_id"}},
	} {
		q := opts.Query.RpcCommandOptions[i+1]
		require.Equal(t, want.method, q.RpcMethod)
		require.Equal(t, want.use, q.Use)
		require.Len(t, q.PositionalArgs, len(want.args))
		for j, arg := range want.args {
			require.Equal(t, arg, q.PositionalArgs[j].ProtoField)
		}
	}

	require.Equal(t, "bze.dao.Msg", opts.Tx.Service)
	require.Len(t, opts.Tx.RpcCommandOptions, 15)
	updateParams := opts.Tx.RpcCommandOptions[0]
	require.Equal(t, "UpdateParams", updateParams.RpcMethod)
	require.Equal(t, "update-params [params]", updateParams.Use)
	require.False(t, updateParams.Skip)
	require.True(t, updateParams.GovProposal)

	createDao := opts.Tx.RpcCommandOptions[1]
	require.Equal(t, "CreateDao", createDao.RpcMethod)
	require.Equal(t, "create-dao [type] [name] [config]", createDao.Use)
	require.False(t, createDao.GovProposal)
	require.Len(t, createDao.PositionalArgs, 3)
	require.Equal(t, "type", createDao.PositionalArgs[0].ProtoField)
	require.Equal(t, "name", createDao.PositionalArgs[1].ProtoField)
	require.Equal(t, "config", createDao.PositionalArgs[2].ProtoField)

	adminExecute := opts.Tx.RpcCommandOptions[2]
	require.Equal(t, "AdminExecute", adminExecute.RpcMethod)
	require.Equal(t, "admin-execute [dao-id] [msgs...]", adminExecute.Use)
	require.False(t, adminExecute.GovProposal)
	require.Len(t, adminExecute.PositionalArgs, 2)
	require.Equal(t, "dao_id", adminExecute.PositionalArgs[0].ProtoField)
	require.Equal(t, "msgs", adminExecute.PositionalArgs[1].ProtoField)
	require.True(t, adminExecute.PositionalArgs[1].Varargs)

	nominateAdmin := opts.Tx.RpcCommandOptions[3]
	require.Equal(t, "NominateAdmin", nominateAdmin.RpcMethod)
	require.Equal(t, "nominate-admin [dao-id] [nominee]", nominateAdmin.Use)
	require.False(t, nominateAdmin.GovProposal)
	require.Len(t, nominateAdmin.PositionalArgs, 2)
	require.Equal(t, "dao_id", nominateAdmin.PositionalArgs[0].ProtoField)
	require.Equal(t, "nominee", nominateAdmin.PositionalArgs[1].ProtoField)

	acceptAdmin := opts.Tx.RpcCommandOptions[4]
	require.Equal(t, "AcceptAdminNomination", acceptAdmin.RpcMethod)
	require.Equal(t, "accept-admin-nomination [dao-id]", acceptAdmin.Use)
	require.False(t, acceptAdmin.GovProposal)
	require.Len(t, acceptAdmin.PositionalArgs, 1)
	require.Equal(t, "dao_id", acceptAdmin.PositionalArgs[0].ProtoField)

	withdrawNomination := opts.Tx.RpcCommandOptions[5]
	require.Equal(t, "WithdrawAdminNomination", withdrawNomination.RpcMethod)
	require.Equal(t, "withdraw-admin-nomination [dao-id]", withdrawNomination.Use)
	require.False(t, withdrawNomination.GovProposal)
	require.Len(t, withdrawNomination.PositionalArgs, 1)
	require.Equal(t, "dao_id", withdrawNomination.PositionalArgs[0].ProtoField)

	submitProposal := opts.Tx.RpcCommandOptions[6]
	require.Equal(t, "SubmitProposal", submitProposal.RpcMethod)
	require.Equal(t, "submit-proposal [dao-id] [title] [msgs...]", submitProposal.Use)
	require.False(t, submitProposal.GovProposal)
	require.Len(t, submitProposal.PositionalArgs, 3)
	require.Equal(t, "dao_id", submitProposal.PositionalArgs[0].ProtoField)
	require.Equal(t, "title", submitProposal.PositionalArgs[1].ProtoField)
	require.Equal(t, "msgs", submitProposal.PositionalArgs[2].ProtoField)
	require.True(t, submitProposal.PositionalArgs[2].Varargs)
	require.Contains(t, submitProposal.FlagOptions, "summary")
	require.Contains(t, submitProposal.FlagOptions, "expedited")

	vote := opts.Tx.RpcCommandOptions[7]
	require.Equal(t, "Vote", vote.RpcMethod)
	require.Equal(t, "vote [proposal-id] [option]", vote.Use)
	require.False(t, vote.GovProposal)
	require.Len(t, vote.PositionalArgs, 2)
	require.Equal(t, "proposal_id", vote.PositionalArgs[0].ProtoField)
	require.Equal(t, "option", vote.PositionalArgs[1].ProtoField)

	cancel := opts.Tx.RpcCommandOptions[8]
	require.Equal(t, "CancelProposal", cancel.RpcMethod)
	require.Equal(t, "cancel-proposal [proposal-id]", cancel.Use)
	require.False(t, cancel.GovProposal)
	require.Len(t, cancel.PositionalArgs, 1)
	require.Equal(t, "proposal_id", cancel.PositionalArgs[0].ProtoField)

	settle := opts.Tx.RpcCommandOptions[9]
	require.Equal(t, "SettleProposal", settle.RpcMethod)
	require.Equal(t, "settle-proposal [proposal-id]", settle.Use)
	require.False(t, settle.GovProposal)
	require.Len(t, settle.PositionalArgs, 1)
	require.Equal(t, "proposal_id", settle.PositionalArgs[0].ProtoField)

	cleanup := opts.Tx.RpcCommandOptions[10]
	require.Equal(t, "CleanupProposal", cleanup.RpcMethod)
	require.Equal(t, "cleanup-proposal [proposal-id]", cleanup.Use)
	require.False(t, cleanup.GovProposal)
	require.Len(t, cleanup.PositionalArgs, 1)
	require.Equal(t, "proposal_id", cleanup.PositionalArgs[0].ProtoField)
	require.Contains(t, cleanup.FlagOptions, "limit")

	updateConfig := opts.Tx.RpcCommandOptions[11]
	require.Equal(t, "UpdateDaoConfig", updateConfig.RpcMethod)
	require.Equal(t, "update-dao-config [dao-id] [config]", updateConfig.Use)
	require.False(t, updateConfig.GovProposal)
	require.Len(t, updateConfig.PositionalArgs, 2)
	require.Equal(t, "dao_id", updateConfig.PositionalArgs[0].ProtoField)
	require.Equal(t, "config", updateConfig.PositionalArgs[1].ProtoField)

	updateMetadata := opts.Tx.RpcCommandOptions[12]
	require.Equal(t, "UpdateMetadata", updateMetadata.RpcMethod)
	require.Equal(t, "update-metadata [dao-id] [name] [description]", updateMetadata.Use)
	require.False(t, updateMetadata.GovProposal)
	require.Len(t, updateMetadata.PositionalArgs, 3)
	require.Equal(t, "dao_id", updateMetadata.PositionalArgs[0].ProtoField)
	require.Equal(t, "name", updateMetadata.PositionalArgs[1].ProtoField)
	require.Equal(t, "description", updateMetadata.PositionalArgs[2].ProtoField)

	updateElectorate := opts.Tx.RpcCommandOptions[13]
	require.Equal(t, "UpdateElectorate", updateElectorate.RpcMethod)
	require.Equal(t, "update-electorate [dao-id] [electorate]", updateElectorate.Use)
	require.False(t, updateElectorate.GovProposal)
	require.Len(t, updateElectorate.PositionalArgs, 2)
	require.Equal(t, "dao_id", updateElectorate.PositionalArgs[0].ProtoField)
	require.Equal(t, "electorate", updateElectorate.PositionalArgs[1].ProtoField)

	// add and remove are repeated fields, so autocli reads them as --add and --remove flags
	updateMembers := opts.Tx.RpcCommandOptions[14]
	require.Equal(t, "UpdateMembers", updateMembers.RpcMethod)
	require.Equal(t, "update-members [dao-id] --add [address,...] --remove [address,...]", updateMembers.Use)
	require.False(t, updateMembers.GovProposal)
	require.Len(t, updateMembers.PositionalArgs, 1)
	require.Equal(t, "dao_id", updateMembers.PositionalArgs[0].ProtoField)
}

// autocli reads the create-dao config argument as protojson into the api type
// and sends its proto bytes; the chain must decode them into a valid DaoConfig.
// Ratios are cosmos.Dec scalars, so raw JSON carries them as 18-decimal
// integers; durations use the protojson form ("259200s").
func TestCreateDaoConfigJSON_DecodesIntoValidConfig(t *testing.T) {
	const configJSON = `{"min_quorum":"500000000000000000","threshold":"500000000000000000",` +
		`"voting_period":"259200s","expedited_period":"86400s",` +
		`"expedite_quorum":"600000000000000000","expedite_threshold":"660000000000000000","proposal_deposit":{"denom":"ubze","amount":"1000"},` +
		`"deposit_refund_policy":"ONLY_PASSED","cost_destination":"TREASURY"}`

	var apiConfig apidao.DaoConfig
	require.NoError(t, protojson.Unmarshal([]byte(configJSON), &apiConfig))
	bz, err := proto.Marshal(&apiConfig)
	require.NoError(t, err)

	var config types.DaoConfig
	require.NoError(t, config.Unmarshal(bz))
	require.NoError(t, config.Validate())
	require.Equal(t, math.LegacyMustNewDecFromStr("0.5"), config.MinQuorum)
	require.Equal(t, math.LegacyMustNewDecFromStr("0.66"), config.ExpediteThreshold)
	require.Equal(t, 72*time.Hour, config.VotingPeriod)
	require.Equal(t, 24*time.Hour, config.ExpeditedPeriod)
	require.Equal(t, types.DepositRefundPolicy_ONLY_PASSED, config.DepositRefundPolicy)
	require.Equal(t, types.FundsDestination_TREASURY, config.CostDestination)
	require.Equal(t, "1000ubze", config.ProposalDeposit.String())
	require.Nil(t, config.ProposalCost)
}

// A human decimal in raw JSON is not a valid cosmos.Dec on the wire.
func TestCreateDaoConfigJSON_RejectsHumanDecimals(t *testing.T) {
	var apiConfig apidao.DaoConfig
	require.NoError(t, protojson.Unmarshal([]byte(`{"min_quorum":"0.5"}`), &apiConfig))
	bz, err := proto.Marshal(&apiConfig)
	require.NoError(t, err)

	var config types.DaoConfig
	require.Error(t, config.Unmarshal(bz))
}
