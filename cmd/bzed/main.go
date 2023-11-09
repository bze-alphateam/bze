package main

import (
	"os"

	"github.com/bze-alphateam/bze/app"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
)

func main() {
	setAddressPrefixes(app.AccountAddressPrefix)
	rootCmd := NewRootCmd(app.MakeEncodingConfig())
	if err := svrcmd.Execute(rootCmd, app.DefaultNodeHome); err != nil {
		os.Exit(1)
	}
}
