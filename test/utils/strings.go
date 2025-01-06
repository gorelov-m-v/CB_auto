package utils

import (
	"math/rand"
	"strings"
	"time"
)

const charset = "abcdefghijklmnopqrstuvwxyz0123456789"

func GenerateAlias() string {
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)
	length := r.Intn(20) + 1
	var sb strings.Builder
	sb.Grow(length)

	for i := 0; i < length; i++ {
		if i > 0 && sb.Len() > 0 && sb.String()[sb.Len()-1] == '-' {
			sb.WriteByte(charset[r.Intn(len(charset))])
		} else {
			if r.Float32() < 0.1 && sb.Len() > 0 && sb.Len() < length-1 {
				sb.WriteByte('-')
			} else {
				sb.WriteByte(charset[r.Intn(len(charset))])
			}
		}
	}

	result := sb.String()
	if result[0] == '-' || result[len(result)-1] == '-' {
		return GenerateAlias()
	}
	return result
}
