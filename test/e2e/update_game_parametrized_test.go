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

const (
	// Типы полей для обновления
	FieldAlias = "alias"
	FieldName  = "name"

	// Префиксы для генерации значений
	AliasPrefix = "updated_game_"
	NamePrefix  = "Updated Game "

	// Значения для запросов к БД
	HasFreeSpinsValue = 1
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

	s.ParamUpdateGame = []UpdateGameParam{
		{
			Alias:       fmt.Sprintf("%s%s", AliasPrefix, utils.Get(utils.ALIAS, 10)),
			Name:        fmt.Sprintf("%s%s", NamePrefix, utils.Get(utils.GAME_TITLE, 10)),
			Description: "Обновление алиаса игры",
		},
		{
			Alias:       fmt.Sprintf("%s%s", AliasPrefix, utils.Get(utils.ALIAS, 10)),
			Name:        fmt.Sprintf("%s%s", NamePrefix, utils.Get(utils.GAME_TITLE, 10)),
			Description: "Обновление названия игры",
		},
	}
}

func (s *ParametrizedUpdateGameSuite) TestAll(t provider.T) {
	for _, param := range s.ParamUpdateGame {
		t.Run(param.Description, func(t provider.T) {
			t.Epic("Games")
			t.Feature("Обновление игры")
			t.Title(fmt.Sprintf("Проверка обновления поля %s", param.Description))
			t.Tags("Games", "Update")

			var gameID string
			// var newValue string
			// var getResp *clientTypes.Response[models.GetCapGamesResponseBody]

			t.WithNewStep("Получение ID игры с free spins из БД", func(sCtx provider.StepCtx) {
				game := s.gameRepo.GetGame(sCtx, map[string]interface{}{
					"has_free_spins": HasFreeSpinsValue,
				})
				sCtx.Require().NotNil(game, "Игра найдена")
				gameID = game.UUID
				sCtx.Require().NotEmpty(gameID, "ID игры получен")
			})

			t.WithNewStep(fmt.Sprintf("Обновление %s игры", param.Alias), func(sCtx provider.StepCtx) {
				updateReq := &clientTypes.Request[models.UpdateCapGamesRequestBody]{

					Headers: map[string]string{
						"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
						"Platform-NodeId": s.config.Node.ProjectID,
					},
					PathParams: map[string]string{
						"id": gameID,
					},
					Body: &models.UpdateCapGamesRequestBody{
						Alias: param.Alias,
						Name:  param.Name,
					},
				}

				updateResp := s.capService.UpdateGames(sCtx, updateReq)
				sCtx.Require().Equal(http.StatusOK, updateResp.StatusCode)
				sCtx.Require().Equal(gameID, updateResp.Body.ID)
			})

			t.WithNewAsyncStep(fmt.Sprintf("Проверка обновления %s в БД и API", param.Description), func(sCtx provider.StepCtx) {
				// Проверка в БД
				gameFromDB := s.gameRepo.GetGame(sCtx, map[string]interface{}{
					"uuid": gameID,
				})
				sCtx.Assert().NotNil(gameFromDB)
				sCtx.Assert().Equal(param.Alias, gameFromDB.Alias)
				sCtx.Assert().Equal(param.Name, gameFromDB.Name)
			})

			t.WithNewAsyncStep("Проверка обновления %s в API", func(sCtx provider.StepCtx) {
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
				sCtx.Assert().Equal(param.Alias, getResp.Body.Alias)
				sCtx.Assert().Equal(param.Name, getResp.Body.Name)
			})

			t.WithNewAsyncStep("Проверка события обновления игры в Kafka", func(sCtx provider.StepCtx) {

				// Ищем сообщение с нужным ID и обновленным значением
				gameMessage := kafka.FindMessageByFilter(sCtx, s.kafka, func(msg kafka.GameMessage) bool {
					return msg.ID == gameID &&
						msg.Alias == param.Alias &&
						msg.Name == param.Name
				})

				sCtx.Assert().NotEmpty(gameMessage, "Сообщение об обновлении игры найдено в Kafka")
				sCtx.Assert().Equal(param.Alias, gameMessage.Alias, "Alias игры в Kafka совпадает с API")
				sCtx.Assert().Equal(param.Name, gameMessage.Name, "Название игры в Kafka совпадает с API")

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
