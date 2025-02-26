package utils

import (
	"fmt"
	"time"
)

func GenerateAlias() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
