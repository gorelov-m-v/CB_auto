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

	_ "github.com/go-sql-driver/mysql" // Импорт драйвера MySQL
	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"
)

type UpdateGameSuite struct {
	suite.Suite
	config     *config.Config
	capService capAPI.CapAPI
	database   *repository.Connector
	gameRepo   *game.Repository
	walletDB   *repository.Connector
}

func (s *UpdateGameSuite) BeforeAll(t provider.T) {
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

		// Подключение к БД wallet для получения игры с free spins
		walletConnector := repository.OpenConnector(t, &s.config.MySQL, repository.Wallet)
		s.walletDB = &walletConnector
	})
}

func (s *UpdateGameSuite) AfterAll(t provider.T) {
	if s.database != nil {
		_ = s.database.Close()
	}
	if s.walletDB != nil {
		_ = s.walletDB.Close()
	}
}

func (s *UpdateGameSuite) TestUpdateGame(t provider.T) {
	var gameID string

	t.WithNewStep("Получение ID игры с free spins из БД", func(sCtx provider.StepCtx) {
		query := "SELECT uuid FROM `beta-09_core`.`game` WHERE has_free_spins = 1 LIMIT 1"
		err := s.database.DB().QueryRowContext(context.Background(), query).Scan(&gameID)
		sCtx.Require().NoError(err, "Ошибка при получении ID игры с free spins")
		sCtx.Require().NotEmpty(gameID, "ID игры получен")
	})

	// Генерируем новый alias и название
	newAlias := "updated_game_" + utils.Get(utils.ALIAS, 10)
	newName := "Updated Game " + utils.Get(utils.GAME_TITLE, 20)

	t.WithNewStep("Обновление alias и названия игры", func(sCtx provider.StepCtx) {
		updateReq := &clientTypes.Request[models.UpdateCapGamesRequestBody]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
				"Platform-NodeId": s.config.Node.ProjectID,
			},
			PathParams: map[string]string{
				"id": gameID,
			},
			Body: &models.UpdateCapGamesRequestBody{
				Alias: newAlias,
				Name:  newName,
			},
		}

		updateResp := s.capService.UpdateGames(sCtx, updateReq)
		sCtx.Assert().Equal(http.StatusOK, updateResp.StatusCode)
		sCtx.Assert().Equal(gameID, updateResp.Body.ID)
	})

	t.WithNewStep("Проверка обновления alias и названия в БД и API", func(sCtx provider.StepCtx) {
		// Проверка в БД
		gameFromDB := s.gameRepo.GetGame(sCtx, map[string]interface{}{
			"uuid": gameID,
		})
		sCtx.Assert().NotNil(gameFromDB)
		sCtx.Assert().Equal(newAlias, gameFromDB.Alias)
		sCtx.Assert().Equal(newName, gameFromDB.Name)

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
		sCtx.Assert().Equal(newAlias, getResp.Body.Alias)
		sCtx.Assert().Equal(newName, getResp.Body.Name)
	})
}

func TestUpdateGameSuite(t *testing.T) {
	suite.RunSuite(t, new(UpdateGameSuite))
}
