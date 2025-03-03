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
	var sb strings.Builder
	var last byte

	// Генерация первого символа
	ch := allowed[rand.Intn(len(allowed))]
	sb.WriteByte(ch)
	last = ch

	// Генерация остальных символов
	for i := 1; i < length; i++ {
		if last == '-' {
			// После дефиса не может быть дефиса
			ch := allowed[:len(allowed)-1][rand.Intn(len(allowed)-1)]
			sb.WriteByte(ch)
			last = ch
		} else {
			ch := allowed[rand.Intn(len(allowed))]
			sb.WriteByte(ch)
			last = ch
		}
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

func GenerateCategoryTitle(length int) string {
	if length > 25 {
		length = 25
	}
	if length <= 0 {
		return ""
	}

	allowed := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"0123456789" +
		"абвгдеёжзийклмнопрстуфхцчшщъыьэюя" +
		"АБВГДЕЁЖЗИЙКЛМНОПРСТУФХЦЧШЩЪЫЬЭЮЯ" +
		" -")

	result := make([]rune, length)
	for i := range result {
		result[i] = allowed[rand.Intn(len(allowed))]
	}

	if result[0] == ' ' {
		result[0] = 'A'
	}
	if result[length-1] == ' ' {
		result[length-1] = 'z'
	}

	return string(result)
}

func GenerateCollectionTitle(length int) string {
	if length > 25 {
		length = 25
	}
	if length <= 0 {
		return ""
	}

	allowed := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"0123456789" +
		"абвгдеёжзийклмнопрстуфхцчшщъыьэюя" +
		"АБВГДЕЁЖЗИЙКЛМНОПРСТУФХЦЧШЩЪЫЬЭЮЯ" +
		" -")

	result := make([]rune, length)
	for i := range result {
		result[i] = allowed[rand.Intn(len(allowed))]
	}

	if result[0] == ' ' {
		result[0] = 'A'
	}
	if result[length-1] == ' ' {
		result[length-1] = 'z'
	}

	return string(result)
}
