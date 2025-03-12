package test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	capAPI "CB_auto/internal/client/cap"
	"CB_auto/internal/client/cap/models"
	"CB_auto/internal/client/factory"
	clientTypes "CB_auto/internal/client/types"
	"CB_auto/internal/config"
	"CB_auto/internal/repository"
	"CB_auto/internal/repository/game"
	"CB_auto/pkg/utils"

	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"
)

type UpdateGameParam struct {
	name        string
	updateField string
}

type ParametrizedUpdateGameSuite struct {
	suite.Suite
	config          *config.Config
	capService      capAPI.CapAPI
	database        *repository.Connector
	gameRepo        *game.Repository
	ParamUpdateGame []UpdateGameParam
}

// Добавляем константы
const (
	DatabaseName = "beta-09_core"
	TableName    = "game"
)

func (s *ParametrizedUpdateGameSuite) BeforeAll(t provider.T) {
	t.WithNewStep("Чтение конфигурационного файла.", func(sCtx provider.StepCtx) {
		s.config = config.ReadConfig(t)
	})

	t.WithNewStep("Инициализация http-клиента и CAP API сервиса.", func(sCtx provider.StepCtx) {
		s.capService = factory.InitClient[capAPI.CapAPI](sCtx, s.config, clientTypes.Cap)
	})

	t.WithNewStep("Подключение к БД", func(sCtx provider.StepCtx) {
		connector := repository.OpenConnector(t, &s.config.MySQL, repository.Core)
		s.database = &connector
		s.gameRepo = game.NewRepository(s.database.DB(), &s.config.MySQL)
	})

	s.ParamUpdateGame = []UpdateGameParam{
		{
			name:        "Обновление alias",
			updateField: "alias",
		},
		{
			name:        "Обновление названия",
			updateField: "name",
		},
	}
}

func (s *ParametrizedUpdateGameSuite) TableTestUpdateGame(t provider.T, param UpdateGameParam) {
	t.Epic("Games")
	t.Feature("Обновление игры")
	t.Title(fmt.Sprintf("Проверка обновления поля %s", param.updateField))
	t.Tags("Games", "Update")

	var gameID string
	var newValue string

	t.WithNewStep("Получение ID игры с free spins из БД", func(sCtx provider.StepCtx) {
		query := fmt.Sprintf("SELECT uuid FROM `%s`.`%s` WHERE has_free_spins = 1 LIMIT 1",
			DatabaseName,
			TableName,
		)
		err := s.database.DB().QueryRowContext(context.Background(), query).Scan(&gameID)
		sCtx.Require().NoError(err, "Ошибка при получении ID игры с free spins")
		sCtx.Require().NotEmpty(gameID, "ID игры получен")
	})

	t.WithNewStep(fmt.Sprintf("Генерация нового значения для поля %s", param.updateField), func(sCtx provider.StepCtx) {
		switch param.updateField {
		case "alias":
			newValue = fmt.Sprintf("updated_game_%s", utils.Get(utils.ALIAS, 10))
		case "name":
			newValue = fmt.Sprintf("Updated Game %s", utils.Get(utils.GAME_TITLE, 10))
		}
		sCtx.Require().NotEmpty(newValue, "Новое значение сгенерировано")
	})

	t.WithNewStep(fmt.Sprintf("Обновление %s игры", param.updateField), func(sCtx provider.StepCtx) {
		updateReq := &clientTypes.Request[models.UpdateCapGamesRequestBody]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
				"Platform-NodeId": s.config.Node.ProjectID,
			},
			PathParams: map[string]string{
				"id": gameID,
			},
			Body: &models.UpdateCapGamesRequestBody{},
		}

		switch param.updateField {
		case "alias":
			updateReq.Body.Alias = newValue
		case "name":
			updateReq.Body.Name = newValue
		}

		updateResp := s.capService.UpdateGames(sCtx, updateReq)
		sCtx.Assert().Equal(http.StatusOK, updateResp.StatusCode)
		sCtx.Assert().Equal(gameID, updateResp.Body.ID)
	})

	t.WithNewStep(fmt.Sprintf("Проверка обновления %s в БД и API", param.updateField), func(sCtx provider.StepCtx) {
		// Проверка в БД
		gameFromDB := s.gameRepo.GetGame(sCtx, map[string]interface{}{
			"uuid": gameID,
		})
		sCtx.Assert().NotNil(gameFromDB)

		switch param.updateField {
		case "alias":
			sCtx.Assert().Equal(newValue, gameFromDB.Alias)
		case "name":
			sCtx.Assert().Equal(newValue, gameFromDB.Name)
		}

		// Проверка через API
		getReq := &clientTypes.Request[struct{}]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
				"Platform-NodeId": s.config.Node.ProjectID,
			},
			PathParams: map[string]string{
				"id": gameID,
			},
		}

		getResp := s.capService.GetGames(sCtx, getReq)
		sCtx.Assert().Equal(http.StatusOK, getResp.StatusCode)
		sCtx.Assert().Equal(gameID, getResp.Body.ID)

		switch param.updateField {
		case "alias":
			sCtx.Assert().Equal(newValue, getResp.Body.Alias)
		case "name":
			sCtx.Assert().Equal(newValue, getResp.Body.Name)
		}
	})
}

func (s *ParametrizedUpdateGameSuite) TestAll(t provider.T) {
	for _, param := range s.ParamUpdateGame {
		t.Run(param.name, func(t provider.T) {
			s.TableTestUpdateGame(t, param)
		})
	}
}

func (s *ParametrizedUpdateGameSuite) AfterAll(t provider.T) {
	if s.database != nil {
		_ = s.database.Close()
	}
}

func TestParametrizedUpdateGameSuite(t *testing.T) {
	suite.RunSuite(t, new(ParametrizedUpdateGameSuite))
}
