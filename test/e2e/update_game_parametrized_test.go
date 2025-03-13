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
	Field       string
	Value       string
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
			Field:       FieldAlias,
			Value:       fmt.Sprintf("%s%s", AliasPrefix, utils.Get(utils.ALIAS, 10)),
			Description: "Обновление алиаса игры",
		},
		{
			Field:       FieldName,
			Value:       fmt.Sprintf("%s%s", NamePrefix, utils.Get(utils.GAME_TITLE, 10)),
			Description: "Обновление названия игры",
		},
	}
}

func (s *ParametrizedUpdateGameSuite) TestAll(t provider.T) {
	for _, param := range s.ParamUpdateGame {
		t.Run(param.Description, func(t provider.T) {
			t.Epic("Games")
			t.Feature("Обновление игры")
			t.Title(fmt.Sprintf("Проверка обновления поля %s", param.Field))
			t.Tags("Games", "Update")

			var gameID string
			var newValue string
			var getResp *clientTypes.Response[models.GetCapGamesResponseBody]

			t.WithNewStep("Получение ID игры с free spins из БД", func(sCtx provider.StepCtx) {
				game := s.gameRepo.GetGame(sCtx, map[string]interface{}{
					"has_free_spins": HasFreeSpinsValue,
				})
				sCtx.Require().NotNil(game, "Игра найдена")
				gameID = game.UUID
				sCtx.Require().NotEmpty(gameID, "ID игры получен")
			})

			t.WithNewStep(fmt.Sprintf("Генерация нового значения для поля %s", param.Field), func(sCtx provider.StepCtx) {
				newValue = param.Value
				sCtx.Require().NotEmpty(newValue, "Новое значение сгенерировано")
			})

			t.WithNewStep(fmt.Sprintf("Обновление %s игры", param.Field), func(sCtx provider.StepCtx) {
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

				switch param.Field {
				case FieldAlias:
					updateReq.Body.Alias = newValue
				case FieldName:
					updateReq.Body.Name = newValue
				}

				updateResp := s.capService.UpdateGames(sCtx, updateReq)
				sCtx.Assert().Equal(http.StatusOK, updateResp.StatusCode)
				sCtx.Assert().Equal(gameID, updateResp.Body.ID)
			})

			t.WithNewStep(fmt.Sprintf("Проверка обновления %s в БД и API", param.Field), func(sCtx provider.StepCtx) {
				// Проверка в БД
				gameFromDB := s.gameRepo.GetGame(sCtx, map[string]interface{}{
					"uuid": gameID,
				})
				sCtx.Assert().NotNil(gameFromDB)

				switch param.Field {
				case FieldAlias:
					sCtx.Assert().Equal(newValue, gameFromDB.Alias)
				case FieldName:
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

				getResp = s.capService.GetGames(sCtx, getReq)
				sCtx.Assert().Equal(http.StatusOK, getResp.StatusCode)
				sCtx.Assert().Equal(gameID, getResp.Body.ID)

				switch param.Field {
				case FieldAlias:
					sCtx.Assert().Equal(newValue, getResp.Body.Alias)
				case FieldName:
					sCtx.Assert().Equal(newValue, getResp.Body.Name)
				}
			})

			t.WithNewAsyncStep("Проверка события обновления игры в Kafka", func(sCtx provider.StepCtx) {
				// Ждем некоторое время, чтобы убедиться, что все сообщения дошли до Kafka

				var expectedValue string
				switch param.Field {
				case FieldAlias:
					expectedValue = getResp.Body.Alias
				case FieldName:
					expectedValue = getResp.Body.Name
				}

				// Ищем сообщение с нужным ID и обновленным значением
				gameMessage := kafka.FindMessageByFilter[kafka.GameMessage](sCtx, s.kafka, func(msg kafka.GameMessage) bool {
					if msg.ID != gameID {
						return false
					}

					switch param.Field {
					case FieldAlias:
						return msg.Alias == expectedValue
					case FieldName:
						return msg.Name == expectedValue
					default:
						return false
					}
				})

				sCtx.Assert().NotEmpty(gameMessage, "Сообщение об обновлении игры найдено в Kafka")

				switch param.Field {
				case FieldAlias:
					sCtx.Assert().Equal(expectedValue, gameMessage.Alias, "Alias игры в Kafka совпадает с API")
				case FieldName:
					sCtx.Assert().Equal(expectedValue, gameMessage.Name, "Название игры в Kafka совпадает с API")
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
