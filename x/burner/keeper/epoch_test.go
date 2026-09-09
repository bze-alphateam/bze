package keeper_test

import (
	"errors"

	"go.uber.org/mock/gomock"
)

func (suite *IntegrationTestSuite) TestGetRaffleCurrentEpoch_Success() {
	suite.epoch.EXPECT().SafeGetEpochCountByIdentifier(suite.ctx, "hour").Return(int64(42), nil).Times(1)

	epoch, err := suite.k.GetRaffleCurrentEpoch(suite.ctx)
	suite.Require().NoError(err)
	suite.Require().Equal(uint64(42), epoch)
}

func (suite *IntegrationTestSuite) TestGetRaffleCurrentEpoch_Error() {
	suite.epoch.EXPECT().SafeGetEpochCountByIdentifier(suite.ctx, "hour").Return(int64(0), errors.New("epoch is catching up")).Times(1)

	_, err := suite.k.GetRaffleCurrentEpoch(suite.ctx)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "catching up")
}

func (suite *IntegrationTestSuite) TestGetRaffleCurrentEpoch_ZeroEpoch() {
	suite.epoch.EXPECT().SafeGetEpochCountByIdentifier(gomock.Any(), "hour").Return(int64(0), nil).Times(1)

	epoch, err := suite.k.GetRaffleCurrentEpoch(suite.ctx)
	suite.Require().NoError(err)
	suite.Require().Equal(uint64(0), epoch)
}
