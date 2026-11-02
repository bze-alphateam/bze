#!/usr/bin/env bash

mockgen_cmd="mockgen"
$mockgen_cmd -source=x/rewards/types/expected_keepers.go -package testutil -destination x/rewards/testutil/expected_keepers_mocks.go
$mockgen_cmd -source=x/burner/types/expected_keepers.go -package testutil -destination x/burner/testutil/expected_keepers_mocks.go
$mockgen_cmd -source=x/cointrunk/types/expected_keepers.go -package testutil -destination x/cointrunk/testutil/expected_keepers_mocks.go
$mockgen_cmd -source=x/tokenfactory/types/expected_keepers.go -package testutil -destination x/tokenfactory/testutil/expected_keepers_mocks.go
$mockgen_cmd -source=x/tradebin/types/expected_keepers.go -package testutil -destination x/tradebin/testutil/expected_keepers_mocks.go
$mockgen_cmd -source=x/txfeecollector/types/expected_keepers.go -package testutil -destination x/txfeecollector/testutil/expected_keepers_mocks.go
$mockgen_cmd -source=x/dao/types/expected_keepers.go -package testutil -destination x/dao/testutil/expected_keepers_mocks.go
$mockgen_cmd -source=x/guardian/types/expected_keepers.go -package testutil -destination x/guardian/testutil/expected_keepers_mocks.go
$mockgen_cmd -source=x/tradebin/migrations/v5/migrator.go -package testutil -destination x/tradebin/testutil/migration_v5_mocks.go -mock_names OrderKeeper=MockV5OrderKeeper,BankKeeper=MockV5BankKeeper
$mockgen_cmd -source=x/tradebin/migrations/v5/params.go -package testutil -destination x/tradebin/testutil/migration_v5_params_mocks.go -mock_names ParamsKeeper=MockV5ParamsKeeper
$mockgen_cmd -source=x/tradebin/keeper/engine.go -package testutil -destination x/tradebin/testutil/engine_mocks.go -mock_names ProcessingKeeper=MockProcessingKeeper,BankKeeper=MockEngineBankKeeper
$mockgen_cmd -source=x/tradebin/migrations/v5/owner_price.go -package testutil -destination x/tradebin/testutil/migration_v5_owner_price_mocks.go -mock_names OwnerPriceKeeper=MockV5OwnerPriceKeeper
$mockgen_cmd -source=x/tradebin/migrations/v5/owner_count.go -package testutil -destination x/tradebin/testutil/migration_v5_owner_count_mocks.go -mock_names PriceLevelOwnersKeeper=MockV5PriceLevelOwnersKeeper
