package test

import (
	"CB_auto/test/config"
	httpClient "CB_auto/test/transport/http"
	publicAPI "CB_auto/test/transport/http/public"
	"CB_auto/test/transport/http/public/models"
	"CB_auto/test/utils"
	"testing"

	"CB_auto/test/transport/nats"

	"github.com/ozontech/allure-go/pkg/allure"
	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"
)

type FastRegistrationSuite struct {
	suite.Suite
	client        *httpClient.Client
	config        *config.Config
	publicService publicAPI.PublicAPI
	natsKV        *nats.KVStore
}

func (s *FastRegistrationSuite) BeforeAll(t provider.T) {
	t.WithNewStep("Чтение конфигурационного файла.", func(sCtx provider.StepCtx) {
		cfg, err := config.ReadConfig()
		if err != nil {
			t.Fatalf("Ошибка при чтении конфигурации: %v", err)
		}
		s.config = cfg
	})

	t.WithNewStep("Инициализация http-клиента и Public API сервиса.", func(sCtx provider.StepCtx) {
		client, err := httpClient.InitClient(s.config, httpClient.Public)
		if err != nil {
			t.Fatalf("InitClient не удался: %v", err)
		}
		s.client = client
		s.publicService = publicAPI.NewPublicClient(s.client)
	})

	t.WithNewStep("Инициализация NATS KV.", func(sCtx provider.StepCtx) {
		kv, err := nats.NewKVStore(&s.config.Nats)
		if err != nil {
			t.Fatalf("NewKVStore не удался: %v", err)
		}
		s.natsKV = kv
	})
}

func (s *FastRegistrationSuite) TestFastRegistration(t provider.T) {
	t.Epic("Users")
	t.Feature("Регистрация пользователя")
	t.Tags("Public", "Users")
	t.Title("Проверка быстрой регистрации пользователя")

	type TestData struct {
		registrationResponse *models.FastRegistrationResponseBody
	}
	var testData TestData

	t.WithNewStep("Быстрая регистрация пользователя.", func(sCtx provider.StepCtx) {
		createReq := &httpClient.Request[models.FastRegistrationRequestBody]{
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: &models.FastRegistrationRequestBody{
				Country:  s.config.Node.DefaultCountry,
				Currency: s.config.Node.DefaultCurrency,
			},
		}

		createResp := s.publicService.FastRegistration(createReq)
		testData.registrationResponse = &createResp.Body

		t.Require().NotEmpty(createResp.Body.Username)
		t.Require().NotEmpty(createResp.Body.Password)

		sCtx.WithAttachments(allure.NewAttachment("FastRegistration Request", allure.JSON, utils.CreateHttpAttachRequest(createReq)))
		sCtx.WithAttachments(allure.NewAttachment("FastRegistration Response", allure.JSON, utils.CreateHttpAttachResponse(createResp)))
	})

	t.WithNewStep("Проверка записи в NATS KV.", func(sCtx provider.StepCtx) {
		key := "003e5b26-5575-4eac-80b4-6dd094978e75"
		value, err := s.natsKV.Get(key)
		if err != nil {
			t.Fatalf("Ошибка при получении значения из KV: %v", err)
		}
		t.Require().NotNil(value, "Значение в KV не найдено")

		sCtx.WithAttachments(allure.NewAttachment("NATS KV Value", allure.JSON, utils.CreatePrettyJSON(string(value))))
	})
}

func (s *FastRegistrationSuite) AfterAll(t provider.T) {
	if s.natsKV != nil {
		s.natsKV.Close()
	}
}

func TestFastRegistrationSuite(t *testing.T) {
	t.Parallel()
	suite.RunSuite(t, new(FastRegistrationSuite))
}
