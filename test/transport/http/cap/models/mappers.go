package models

const (
	StatusEnabled  = 1
	StatusDisabled = 2
)

type BrandMapper struct{}

func NewBrandMapper() *BrandMapper {
	return &BrandMapper{}
}

func (m *BrandMapper) MapStatusToEnabled(status int) bool {
	return status == StatusEnabled
}

func (m *BrandMapper) MapEnabledToStatus(enabled bool) int {
	if enabled {
		return StatusEnabled
	}
	return StatusDisabled
}
