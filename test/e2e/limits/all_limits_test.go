package test

import (
	"testing"

	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"
)

type AllLimitsSuite struct {
	suite.Suite
}

func (s *AllLimitsSuite) TestCasinoLossLimit(t provider.T) {
	t.Parallel()
	s.RunSuite(t, new(CasinoLossLimitSuite))
}

func (s *AllLimitsSuite) TestSingleBetLimit(t provider.T) {
	t.Parallel()
	s.RunSuite(t, new(SingleBetLimitSuite))
}

func (s *AllLimitsSuite) TestTurnoverLimit(t provider.T) {
	t.Parallel()
	s.RunSuite(t, new(TurnoverLimitSuite))
}

func TestAllLimits(t *testing.T) {
	t.Parallel()
	suite.RunSuite(t, new(AllLimitsSuite))
}
