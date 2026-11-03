package app_test

import (
	"testing"

	"cosmossdk.io/client/v2/autocli"
	"cosmossdk.io/depinject"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/app"
)

// The bzed CLI builds its root command from app.AppConfig() alone (no app.New), so every provider a module needs
// must live in AppConfig(); otherwise every bzed command, even `bzed version`, panics at startup. The client context
// is supplied empty here: the root command builds it from the node home, which a test must not touch.
func TestAppConfig_ResolvesStandaloneLikeTheRootCmd(t *testing.T) {
	var (
		autoCliOpts        autocli.AppOptions
		moduleBasicManager module.BasicManager
	)

	err := depinject.Inject(
		depinject.Configs(app.AppConfig(), depinject.Supply(log.NewNopLogger(), client.Context{})),
		&autoCliOpts,
		&moduleBasicManager,
	)
	require.NoError(t, err)
	require.NotEmpty(t, moduleBasicManager)
	require.NotEmpty(t, autoCliOpts.Modules)
}
