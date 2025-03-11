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

	_ "github.com/go-sql-driver/mysql" // Импорт драйвера MySQL
	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"
	"github.com/stretchr/testify/require"
)

type UpdateGameSuite struct {
	suite.Suite
	config     *config.Config
	capService capAPI.CapAPI
	database   *repository.Connector
	gameRepo   *game.Repository
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
	})
}

func (s *UpdateGameSuite) AfterAll(t provider.T) {
	if s.database != nil {
		_ = s.database.Close()
	}
}

func (s *UpdateGameSuite) TestUpdateGame(t provider.T) {
	gameID := "366a3ad8-7f21-4051-b398-44b54b8a8dff"
	nodeID := "4c59ecfb-9571-4d2e-8e8b-4558636049fc"

	// Шаг 1: Проверка существования игры и соответствия ID
	t.WithNewStep("Проверка существования игры в БД и соответствия ID", func(sCtx provider.StepCtx) {
		// Получаем игру из БД
		gameFromDB := s.gameRepo.GetGame(sCtx, map[string]interface{}{
			"uuid": gameID,
		})
		require.NotNil(sCtx, gameFromDB, "Игра найдена в БД")
		sCtx.Assert().Equal(gameID, gameFromDB.UUID, "UUID игры в БД соответствует ожидаемому")

		// Получаем игру через API
		req := &clientTypes.Request[struct{}]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
				"Platform-NodeId": nodeID,
			},
			PathParams: map[string]string{
				"id": gameID,
			},
		}

		resp := s.capService.GetGames(sCtx, req)
		sCtx.Assert().Equal(http.StatusOK, resp.StatusCode, "Игра успешно получена через API")
		sCtx.Assert().Equal(gameID, resp.Body.ID, "ID игры в ответе API соответствует ожидаемому")

		// Проверяем соответствие данных из БД и API
		sCtx.Assert().Equal(gameFromDB.Alias, resp.Body.Alias, "Alias игры в БД соответствует Alias в API")
		sCtx.Assert().Equal(gameFromDB.Name, resp.Body.Name, "Name игры в БД соответствует Name в API")
	})

	// Шаг 2: Изменение названия игры
	newName := "Обновленное Название Игры " + utils.Get(utils.LETTERS, 5)

	t.WithNewStep("Изменение названия игры", func(sCtx provider.StepCtx) {
		// Получаем текущие данные игры для сохранения остальных полей
		getReq := &clientTypes.Request[struct{}]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
				"Platform-NodeId": nodeID,
			},
			PathParams: map[string]string{
				"id": gameID,
			},
		}

		getResp := s.capService.GetGames(sCtx, getReq)
		sCtx.Assert().Equal(http.StatusOK, getResp.StatusCode, "Игра успешно получена через API")

		// Сохраняем текущий alias
		currentAlias := getResp.Body.Alias

		// Обновляем название игры
		updateBody := models.UpdateCapGamesRequestBody{
			Alias: currentAlias, // Сохраняем текущий alias
			Name:  newName,      // Обновляем название
			// Не указываем Image и RuleResource, так как они пустые
		}

		updateReq := &clientTypes.Request[models.UpdateCapGamesRequestBody]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
				"Platform-NodeId": nodeID,
			},
			PathParams: map[string]string{
				"id": gameID,
			},
			Body: &updateBody,
		}

		updateResp := s.capService.UpdateGames(sCtx, updateReq)
		sCtx.Assert().Equal(http.StatusOK, updateResp.StatusCode, "Игра успешно обновлена")
		sCtx.Assert().Equal(gameID, updateResp.Body.ID, "ID игры в ответе API соответствует ожидаемому")
	})

	// Шаг 3: Проверка изменения названия
	t.WithNewStep("Проверка изменения названия игры", func(sCtx provider.StepCtx) {
		// Получаем обновленную игру из БД
		gameFromDB := s.gameRepo.GetGame(sCtx, map[string]interface{}{
			"uuid": gameID,
		})
		require.NotNil(sCtx, gameFromDB, "Игра найдена в БД после обновления")

		// Получаем обновленную игру через API
		req := &clientTypes.Request[struct{}]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
				"Platform-NodeId": nodeID,
			},
			PathParams: map[string]string{
				"id": gameID,
			},
		}

		resp := s.capService.GetGames(sCtx, req)
		sCtx.Assert().Equal(http.StatusOK, resp.StatusCode, "Игра успешно получена через API после обновления")

		// Проверяем, что название изменилось и совпадает в БД и API
		sCtx.Assert().Equal(newName, resp.Body.Name, "Название игры в API обновлено")
		sCtx.Assert().Equal(gameFromDB.Name, resp.Body.Name, "Название игры в БД соответствует названию в API после обновления")
	})
}

func TestUpdateGameSuite(t *testing.T) {
	suite.RunSuite(t, new(UpdateGameSuite))
}
