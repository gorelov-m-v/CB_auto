package utils

import (
	"math/rand"
	"strings"
)

func GenerateAlias(length int) string {
	if length > 100 {
		length = 100
	}
	if length <= 0 {
		return ""
	}
	allowed := "abcdefghijklmnopqrstuvwxyz0123456789-"
	allowedNoHyphen := "abcdefghijklmnopqrstuvwxyz0123456789"
	var sb strings.Builder
	var last byte
	for i := 0; i < length; i++ {
		var choices string
		if i > 0 && last == '-' {
			choices = allowedNoHyphen
		} else {
			choices = allowed
		}
		ch := choices[rand.Intn(len(choices))]
		sb.WriteByte(ch)
		last = ch
	}
	return sb.String()
}

func GenerateBrandTitle(length int) string {
	if length > 100 {
		length = 100
	}
	if length <= 0 {
		return ""
	}

	allowed := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz" +
		"0123456789" +
		"абвгдеёжзийклмнопрстуфхцчшщъыьэюя" +
		"АБВГДЕЁЖЗИЙКЛМНОПРСТУФХЦЧШЩЪЫЬЭЮЯ" +
		" .,;:!?-_"
	var result string

	for i := 0; i < 10; i++ {
		result = randomStringFromSet(allowed, length)
		if strings.TrimSpace(result) != "" {
			return result
		}
	}

	return "A" + result[1:]
}
