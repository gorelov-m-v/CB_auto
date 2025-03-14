package test

import (
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

	"CB_auto/internal/transport/kafka"

	_ "github.com/go-sql-driver/mysql" // Импорт драйвера MySQL

	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"
)

type UpdateGameParam struct {
	Alias       string
	Name        string
	Description string
}

type ParametrizedUpdateGameSuite struct {
	suite.Suite
	config          *config.Config
	capService      capAPI.CapAPI
	database        *repository.Connector
	gameRepo        *game.Repository
	kafka           *kafka.Kafka
	ParamUpdateGame []UpdateGameParam
}

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

	t.WithNewStep("Инициализация Kafka", func(sCtx provider.StepCtx) {
		s.kafka = kafka.GetInstance(t, s.config)
	})

	// Все параметры, включая новые кейсы
	s.ParamUpdateGame = []UpdateGameParam{
		{
			Alias:       utils.Get(utils.ALIAS, 2),
			Name:        utils.Get(utils.GAME_TITLE, 2),
			Description: "Обновление алиаса и названия игры (граничное минимальное значение)",
		},
		{
			Alias:       utils.Get(utils.ALIAS, 100),
			Name:        utils.Get(utils.GAME_TITLE, 255),
			Description: "Обновление алиаса и названия игры (граничное максимальное значение)",
		},
		{
			Alias:       utils.Get(utils.ALIAS, 99),
			Name:        utils.Get(utils.GAME_TITLE, 254),
			Description: "Обновление алиаса и названия игры (максимальное значение)",
		},
		{
			Alias:       utils.Get(utils.ALIAS, 100),
			Name:        utils.Get(utils.GAME_TITLE, 255),
			Description: "Обновление названия игры",
		},
		{
			Alias:       utils.Get(utils.ALIAS, 2),
			Description: "Обновление только Alias игры (минимальное значение)",
		},
		{
			Name:        utils.Get(utils.GAME_TITLE, 2),
			Description: "Обновление только Name игры (минимальное значение)",
		},
		{
			Name:        utils.Get(utils.GAME_TITLE, 255),
			Description: "Обновление только Name игры (максимальное значение)",
		},
		{
			Alias:       utils.Get(utils.ALIAS, 100),
			Description: "Обновление только Alias игры (граничное максимальное значение)",
		},
		{
			Name:        utils.Get(utils.GAME_TITLE, 254),
			Description: "Обновление только Name игры (граничное максимальное значение)",
		},
		{
			Alias:       utils.Get(utils.ALIAS, 99),
			Description: "Обновление только Alias игры (граничное максимальное значение)",
		},
	}
}

func (s *ParametrizedUpdateGameSuite) TestAll(t provider.T) {
	for _, param := range s.ParamUpdateGame {
		t.Run(param.Description, func(t provider.T) {
			t.Epic("Games")
			t.Feature("Обновление игры")
			t.Title(fmt.Sprintf("Проверка обновления: %s", param.Description))
			t.Tags("Games", "Update")

			var testData struct {
				gameID string
			}

			t.WithNewStep("Получение ID игры с free spins из БД", func(sCtx provider.StepCtx) {
				game := s.gameRepo.GetGame(sCtx, map[string]interface{}{
					"has_free_spins": 1,
				})
				sCtx.Require().NotNil(game, "Игра найдена")
				testData.gameID = game.UUID
				sCtx.Require().NotEmpty(testData.gameID, "ID игры получен")
			})

			t.WithNewStep(fmt.Sprintf("Обновление игры: %s", param.Description), func(sCtx provider.StepCtx) {
				body := &models.UpdateCapGamesRequestBody{}
				if param.Alias != "" {
					body.Alias = param.Alias
				}
				if param.Name != "" {
					body.Name = param.Name
				}

				req := &clientTypes.Request[models.UpdateCapGamesRequestBody]{
					Headers: map[string]string{
						"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
						"Platform-NodeId": s.config.Node.ProjectID,
					},
					PathParams: map[string]string{
						"id": testData.gameID,
					},
					Body: body,
				}

				resp := s.capService.UpdateGames(sCtx, req)
				sCtx.Require().Equal(http.StatusOK, resp.StatusCode)
				sCtx.Require().Equal(testData.gameID, resp.Body.ID)
			})

			t.WithNewAsyncStep(fmt.Sprintf("Проверка обновления %s в БД", param.Description), func(sCtx provider.StepCtx) {
				gameFromDB := s.gameRepo.GetGame(sCtx, map[string]interface{}{
					"uuid": testData.gameID,
				})
				sCtx.Assert().NotNil(gameFromDB)

				if param.Alias != "" {
					sCtx.Assert().Equal(param.Alias, gameFromDB.Alias, "Alias совпадает в БД")
				}
				if param.Name != "" {
					sCtx.Assert().Equal(param.Name, gameFromDB.Name, "Name совпадает в БД")
				}
			})

			t.WithNewAsyncStep(fmt.Sprintf("Проверка обновления %s в API", param.Description), func(sCtx provider.StepCtx) {
				getReq := &clientTypes.Request[struct{}]{
					Headers: map[string]string{
						"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
						"Platform-NodeId": s.config.Node.ProjectID,
					},
					PathParams: map[string]string{
						"id": testData.gameID,
					},
				}

				getResp := s.capService.GetGames(sCtx, getReq)
				sCtx.Assert().Equal(http.StatusOK, getResp.StatusCode)
				sCtx.Assert().Equal(testData.gameID, getResp.Body.ID)

				if param.Alias != "" {
					sCtx.Assert().Equal(param.Alias, getResp.Body.Alias, "Alias совпадает в API")
				}
				if param.Name != "" {
					sCtx.Assert().Equal(param.Name, getResp.Body.Name, "Name совпадает в API")
				}
			})

			t.WithNewAsyncStep(fmt.Sprintf("Проверка события Kafka для %s", param.Description), func(sCtx provider.StepCtx) {
				gameMessage := kafka.FindMessageByFilter(sCtx, s.kafka, func(msg kafka.GameMessage) bool {
					match := msg.ID == testData.gameID
					if param.Alias != "" {
						match = match && msg.Alias == param.Alias
					}
					if param.Name != "" {
						match = match && msg.Name == param.Name
					}
					return match
				})

				sCtx.Assert().NotEmpty(gameMessage, "Сообщение об обновлении игры найдено в Kafka")
				if param.Alias != "" {
					sCtx.Assert().Equal(param.Alias, gameMessage.Alias, "Alias совпадает в Kafka")
				}
				if param.Name != "" {
					sCtx.Assert().Equal(param.Name, gameMessage.Name, "Name совпадает в Kafka")
				}
			})
		})
	}
}

func (s *ParametrizedUpdateGameSuite) AfterAll(t provider.T) {
	if s.database != nil {
		_ = s.database.Close()
	}
	kafka.CloseInstance(t)
}

func TestParametrizedUpdateGameSuite(t *testing.T) {
	suite.RunSuite(t, new(ParametrizedUpdateGameSuite))
}
