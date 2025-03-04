package mappers

import (
	"strconv"

	"CB_auto/internal/client/cap/models"
	"CB_auto/internal/transport/nats"
)

var (
	natsReasonToCapReason = map[int]models.ReasonType{
		1: models.ReasonOperationalMistake,
		2: models.ReasonMalfunction,
		3: models.ReasonBalanceCorrection,
	}

	natsOperationTypeToCapType = map[int]models.OperationType{
		1: models.OperationTypeDeposit,
		2: models.OperationTypeWithdrawal,
		3: models.OperationTypeGift,
		4: models.OperationTypeReferralCommission,
		5: models.OperationTypeCashback,
		6: models.OperationTypeTournamentPrize,
		7: models.OperationTypeJackpot,
	}

	natsDirectionToCapDirection = map[int]models.DirectionType{
		1: models.DirectionIncrease,
		2: models.DirectionDecrease,
	}

	natsLimitEventTypeToCapEventType = map[string]string{
		nats.EventTypeCreated: "created",
		nats.EventTypeUpdated: "updated",
		nats.EventTypeDeleted: "deleted",
	}

	natsLimitTypeToCapLimitType = map[string]models.LimitType{
		nats.LimitTypeSingleBet:     models.LimitTypeSingleBet,
		nats.LimitTypeCasinoLoss:    models.LimitTypeCasinoLoss,
		nats.LimitTypeTurnoverFunds: models.LimitTypeTurnover,
	}

	natsLimitIntervalToCapInterval = map[string]models.LimitPeriodType{
		nats.IntervalTypeDaily:   models.LimitPeriodDaily,
		nats.IntervalTypeWeekly:  models.LimitPeriodWeekly,
		nats.IntervalTypeMonthly: models.LimitPeriodMonthly,
	}
)

func MapReasonFromNats(reason int) models.ReasonType {
	if mapped, ok := natsReasonToCapReason[reason]; ok {
		return mapped
	}
	return models.ReasonBalanceCorrection
}

func MapReasonToNats(reason models.ReasonType) int {
	for k, v := range natsReasonToCapReason {
		if v == reason {
			return k
		}
	}
	return 3
}

func MapOperationTypeFromNats(opType int) models.OperationType {
	if mapped, ok := natsOperationTypeToCapType[opType]; ok {
		return mapped
	}
	return models.OperationTypeCorrection
}

func MapOperationTypeToNats(opType models.OperationType) int {
	for k, v := range natsOperationTypeToCapType {
		if v == opType {
			return k
		}
	}
	return 0
}

func MapDirectionFromNats(direction int) models.DirectionType {
	if mapped, ok := natsDirectionToCapDirection[direction]; ok {
		return mapped
	}
	return models.DirectionIncrease
}

func MapDirectionToNats(direction models.DirectionType) int {
	for k, v := range natsDirectionToCapDirection {
		if v == direction {
			return k
		}
	}
	return 1
}

func MapLimitTypeFromNats(limitType string) models.LimitType {
	if mapped, ok := natsLimitTypeToCapLimitType[limitType]; ok {
		return mapped
	}
	return models.LimitTypeSingleBet
}

func MapLimitIntervalFromNats(interval string) models.LimitPeriodType {
	if mapped, ok := natsLimitIntervalToCapInterval[interval]; ok {
		return mapped
	}
	return models.LimitPeriodDaily
}

func AmountToString(amount float64) string {
	return strconv.FormatFloat(amount, 'f', 2, 64)
}

func StringToAmount(amount string) float64 {
	value, _ := strconv.ParseFloat(amount, 64)
	return value
}

func MapNatsBalanceAdjustmentToCapModel(natsPayload nats.BalanceAdjustedPayload) models.CreateBalanceAdjustmentRequestBody {
	amount, _ := strconv.ParseFloat(natsPayload.Amount, 64)

	return models.CreateBalanceAdjustmentRequestBody{
		Currency:      natsPayload.Currenc,
		Amount:        amount,
		Reason:        mapReasonFromNats(natsPayload.Reason),
		OperationType: mapOperationTypeFromNats(natsPayload.OperationType),
		Direction:     mapDirectionFromNats(natsPayload.Direction),
		Comment:       natsPayload.Comment,
	}
}

func MapNatsLimitToCapLimit(natsLimit nats.LimitChangedV2) []models.PlayerLimit {
	var capLimits []models.PlayerLimit

	for _, limit := range natsLimit.Limits {
		capLimits = append(capLimits, models.PlayerLimit{
			Type:      mapLimitTypeFromNats(limit.LimitType),
			Status:    limit.Status,
			Period:    mapLimitIntervalFromNats(limit.IntervalType),
			Currency:  limit.CurrencyCode,
			Amount:    limit.Amount,
			StartedAt: limit.StartedAt,
			ExpiresAt: limit.ExpiresAt,
		})
	}

	return capLimits
}

func mapReasonFromNats(reason int) models.ReasonType {
	if mapped, ok := natsReasonToCapReason[reason]; ok {
		return mapped
	}
	return models.ReasonBalanceCorrection
}

func mapOperationTypeFromNats(opType int) models.OperationType {
	if mapped, ok := natsOperationTypeToCapType[opType]; ok {
		return mapped
	}
	return models.OperationTypeCorrection
}

func mapDirectionFromNats(direction int) models.DirectionType {
	if mapped, ok := natsDirectionToCapDirection[direction]; ok {
		return mapped
	}
	return models.DirectionIncrease
}

func mapLimitTypeFromNats(limitType string) models.LimitType {
	if mapped, ok := natsLimitTypeToCapLimitType[limitType]; ok {
		return mapped
	}
	return models.LimitTypeSingleBet
}

func mapLimitIntervalFromNats(interval string) models.LimitPeriodType {
	if mapped, ok := natsLimitIntervalToCapInterval[interval]; ok {
		return mapped
	}
	return models.LimitPeriodDaily
}

func MapCapModelToNatsBalanceAdjustment(capModel models.CreateBalanceAdjustmentRequestBody) nats.BalanceAdjustedPayload {
	return nats.BalanceAdjustedPayload{
		Currenc:       capModel.Currency,
		Amount:        strconv.FormatFloat(capModel.Amount, 'f', 2, 64),
		Reason:        mapReasonToNats(capModel.Reason),
		OperationType: mapOperationTypeToNats(capModel.OperationType),
		Direction:     mapDirectionToNats(capModel.Direction),
		Comment:       capModel.Comment,
	}
}

func mapReasonToNats(reason models.ReasonType) int {
	for k, v := range natsReasonToCapReason {
		if v == reason {
			return k
		}
	}
	return 0
}

func mapOperationTypeToNats(opType models.OperationType) int {
	for k, v := range natsOperationTypeToCapType {
		if v == opType {
			return k
		}
	}
	return 0
}

func mapDirectionToNats(direction models.DirectionType) int {
	for k, v := range natsDirectionToCapDirection {
		if v == direction {
			return k
		}
	}
	return 1
}
