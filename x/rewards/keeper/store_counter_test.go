package keeper_test

func (suite *IntegrationTestSuite) TestGetAndSetCounter() {
	key := []byte("test-counter")

	// Test initial counter value (should be 0)
	counter := suite.k.GetCounter(suite.ctx, key)
	suite.Require().Equal(uint64(0), counter)

	// Test SetCounter
	suite.k.SetCounter(suite.ctx, key, 100)

	// Test GetCounter
	retrievedCounter := suite.k.GetCounter(suite.ctx, key)
	suite.Require().Equal(uint64(100), retrievedCounter)
}

func (suite *IntegrationTestSuite) TestSetCounter_MultipleKeys() {
	key1 := []byte("counter1")
	key2 := []byte("counter2")

	// Set different values for different keys
	suite.k.SetCounter(suite.ctx, key1, 50)
	suite.k.SetCounter(suite.ctx, key2, 150)

	// Verify both counters are stored correctly
	counter1 := suite.k.GetCounter(suite.ctx, key1)
	counter2 := suite.k.GetCounter(suite.ctx, key2)

	suite.Require().Equal(uint64(50), counter1)
	suite.Require().Equal(uint64(150), counter2)
}

func (suite *IntegrationTestSuite) TestSetCounter_UpdateExisting() {
	key := []byte("update-test")

	// Set initial value
	suite.k.SetCounter(suite.ctx, key, 10)
	counter := suite.k.GetCounter(suite.ctx, key)
	suite.Require().Equal(uint64(10), counter)

	// Update value
	suite.k.SetCounter(suite.ctx, key, 25)
	counter = suite.k.GetCounter(suite.ctx, key)
	suite.Require().Equal(uint64(25), counter)
}

func (suite *IntegrationTestSuite) TestGetCounter_NonExistentKey() {
	nonExistentKey := []byte("does-not-exist")

	// Should return 0 for non-existent key
	counter := suite.k.GetCounter(suite.ctx, nonExistentKey)
	suite.Require().Equal(uint64(0), counter)
}

func (suite *IntegrationTestSuite) TestSetCounter_EdgeValues() {
	key := []byte("edge-values")

	// Test zero value
	suite.k.SetCounter(suite.ctx, key, 0)
	counter := suite.k.GetCounter(suite.ctx, key)
	suite.Require().Equal(uint64(0), counter)

	// Test maximum uint64 value
	maxValue := uint64(18446744073709551615)
	suite.k.SetCounter(suite.ctx, key, maxValue)
	counter = suite.k.GetCounter(suite.ctx, key)
	suite.Require().Equal(maxValue, counter)
}

func (suite *IntegrationTestSuite) TestGetAndSetStakingRewardsCounter() {
	// Test initial staking rewards counter
	counter := suite.k.GetStakingRewardsCounter(suite.ctx)
	suite.Require().Equal(uint64(0), counter)

	// Test SetStakingRewardsCounter
	suite.k.SetStakingRewardsCounter(suite.ctx, 42)

	// Test GetStakingRewardsCounter
	retrievedCounter := suite.k.GetStakingRewardsCounter(suite.ctx)
	suite.Require().Equal(uint64(42), retrievedCounter)
}

func (suite *IntegrationTestSuite) TestStakingRewardsCounter_MultipleUpdates() {
	// Test multiple sequential updates
	values := []uint64{1, 10, 100, 1000}

	for _, value := range values {
		suite.k.SetStakingRewardsCounter(suite.ctx, value)
		counter := suite.k.GetStakingRewardsCounter(suite.ctx)
		suite.Require().Equal(value, counter)
	}
}

func (suite *IntegrationTestSuite) TestGetAndSetTradingRewardsCounter() {
	// Test initial trading rewards counter
	counter := suite.k.GetTradingRewardsCounter(suite.ctx)
	suite.Require().Equal(uint64(0), counter)

	// Test SetTradingRewardsCounter
	suite.k.SetTradingRewardsCounter(suite.ctx, 99)

	// Test GetTradingRewardsCounter
	retrievedCounter := suite.k.GetTradingRewardsCounter(suite.ctx)
	suite.Require().Equal(uint64(99), retrievedCounter)
}

func (suite *IntegrationTestSuite) TestTradingRewardsCounter_MultipleUpdates() {
	// Test multiple sequential updates
	values := []uint64{5, 50, 500, 5000}

	for _, value := range values {
		suite.k.SetTradingRewardsCounter(suite.ctx, value)
		counter := suite.k.GetTradingRewardsCounter(suite.ctx)
		suite.Require().Equal(value, counter)
	}
}

func (suite *IntegrationTestSuite) TestCounters_Independence() {
	// Test that staking and trading counters are independent
	suite.k.SetStakingRewardsCounter(suite.ctx, 123)
	suite.k.SetTradingRewardsCounter(suite.ctx, 456)

	stakingCounter := suite.k.GetStakingRewardsCounter(suite.ctx)
	tradingCounter := suite.k.GetTradingRewardsCounter(suite.ctx)

	suite.Require().Equal(uint64(123), stakingCounter)
	suite.Require().Equal(uint64(456), tradingCounter)

	// Update one, verify the other is unchanged
	suite.k.SetStakingRewardsCounter(suite.ctx, 999)

	stakingCounter = suite.k.GetStakingRewardsCounter(suite.ctx)
	tradingCounter = suite.k.GetTradingRewardsCounter(suite.ctx)

	suite.Require().Equal(uint64(999), stakingCounter)
	suite.Require().Equal(uint64(456), tradingCounter) // Should be unchanged
}

func (suite *IntegrationTestSuite) TestSetCounter_LargeKey() {
	// Test with larger key
	largeKey := []byte("this-is-a-very-long-key-name-for-testing-purposes-with-many-characters")

	suite.k.SetCounter(suite.ctx, largeKey, 888)
	counter := suite.k.GetCounter(suite.ctx, largeKey)
	suite.Require().Equal(uint64(888), counter)
}
