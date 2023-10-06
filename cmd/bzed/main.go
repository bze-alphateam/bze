package main

import (
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/ibc-go/v7/testing/simapp/simd/cmd"
	"os"

	"github.com/bze-alphateam/bze/app"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
)

func main() {
	//	app.SetAddressPrefixes() //TODO: either implement or remove
	rootCmd, _ := cmd.NewRootCmd()

	if err := svrcmd.Execute(rootCmd, "bzed", app.DefaultNodeHome); err != nil {
		switch e := err.(type) {
		case server.ErrorCode:
			os.Exit(e.Code)

		default:
			os.Exit(1)
		}
	}
}
