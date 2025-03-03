package utils

import (
	"errors"
	"math/rand"
	"strings"
	"time"
	"unicode"
)

const (
	INTEGER           = "integer"
	LETTERS           = "letters"
	ALPHANUMERIC      = "alphanum"
	NUMBER            = "number"
	NAME              = "name"
	EMAIL             = "email"
	PASSWORD          = "pass"
	BIRTHDAY_DDMMYYYY = "birthday_ddmmyyyy"
	BIRTHDAY_YYYYMMDD = "birthday_yyyymmdd"
	CYRILLIC          = "cyrillic"
	SPECIAL           = "special"
	HEX               = "hex"
	NON_HEX           = "non_hex"
	IBAN              = "iban"
	PERSONAL_ID       = "personal_id"
	BRAND_TITLE       = "brandTitle"
	ALIAS             = "alias"
	CATEGORY_TITLE    = "category_title"
)

const (
	CONSONANTS       = "bcdfghjklmnpqrstvwxz"
	VOWELS           = "aeiouy"
	CYRILLIC_LETTERS = "абвгдеёжэийклмнопрстуфхцчшщъыьэюя"
	SPECIAL_CHARS    = "!@#$%^&_"
	HEX_CHARS        = "0123456789abcdef"
	NON_HEX_CHARS    = "ghijklmnopqrstuvwxyz"
)

var (
	latinChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	digits     = "0123456789"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func Get(config string, length int) string {
	switch config {
	case INTEGER:
		return randomNumericString(length)
	case LETTERS:
		return randomLettersString(length)
	case ALPHANUMERIC:
		return randomAlphaNumericString(length)
	case NUMBER:
		s := randomNumericString(length)
		if len(s) > 0 && s[0] == '0' {
			s = string(randomNonZeroDigit()) + s[1:]
		}
		return s
	case NAME:
		return generateName(length)
	case EMAIL:
		return randomAlphaNumericString(length) + "@gmail.com"
	case PASSWORD:
		p, err := generatePassword(length)
		if err != nil {
			return err.Error()
		}
		return p
	case BIRTHDAY_DDMMYYYY:
		return generateBirthday(length, "02.01.2006")
	case BIRTHDAY_YYYYMMDD:
		return generateBirthday(length, "2006-01-02")
	case CYRILLIC:
		return randomStringFromSet(CYRILLIC_LETTERS, length)
	case SPECIAL:
		return randomStringFromSet(SPECIAL_CHARS, length)
	case HEX:
		return randomStringFromSet(HEX_CHARS, length)
	case NON_HEX:
		return randomStringFromSet(NON_HEX_CHARS, length)
	case IBAN:
		return generateIban()
	case PERSONAL_ID:
		return generatePersonalId()
	case BRAND_TITLE:
		return generateRandomString(length, latinChars+digits)
	case ALIAS:
		return GenerateAlias(length)
	case CATEGORY_TITLE:
		return generateCategoryTitle(length)
	default:
		return "Unknown argument to perform random string generation!"
	}
}

func randomNumericString(length int) string {
	digits := "0123456789"
	return randomStringFromSet(digits, length)
}

func randomLettersString(length int) string {
	letters := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	return randomStringFromSet(letters, length)
}

func randomAlphaNumericString(length int) string {
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	return randomStringFromSet(chars, length)
}

func randomStringFromSet(charset string, length int) string {
	var sb strings.Builder
	for i := 0; i < length; i++ {
		sb.WriteByte(charset[rand.Intn(len(charset))])
	}
	return sb.String()
}

func randomNonZeroDigit() byte {
	digits := "123456789"
	return digits[rand.Intn(len(digits))]
}

func generateName(nameLength int) string {
	if nameLength <= 0 {
		return ""
	}
	var sb strings.Builder
	first := CONSONANTS[rand.Intn(len(CONSONANTS))]
	sb.WriteRune(unicode.ToUpper(rune(first)))
	for i := 1; i < nameLength; i++ {
		if i%2 == 0 {
			sb.WriteByte(CONSONANTS[rand.Intn(len(CONSONANTS))])
		} else {
			sb.WriteByte(VOWELS[rand.Intn(len(VOWELS))])
		}
	}
	return sb.String()
}

func generatePassword(passLength int) (string, error) {
	if passLength < 4 {
		return "", errors.New("Password length must be 4 or more!")
	}
	specialSet := "!:@#$%"
	specialChar := string(specialSet[rand.Intn(len(specialSet))])
	numericChar := string("0123456789"[rand.Intn(10)])
	lettersPart := ""
	for lettersPart == "" {
		candidate := randomLettersString(passLength - 2)
		if hasUpperAndLower(candidate) {
			lettersPart = candidate
		}
	}
	combined := []rune(lettersPart + specialChar + numericChar)
	shuffleRunes(combined)
	return string(combined), nil
}

func hasUpperAndLower(s string) bool {
	hasUpper := false
	hasLower := false
	for _, r := range s {
		if unicode.IsUpper(r) {
			hasUpper = true
		}
		if unicode.IsLower(r) {
			hasLower = true
		}
	}
	return hasUpper && hasLower
}

func shuffleRunes(runes []rune) {
	for i := range runes {
		j := rand.Intn(i + 1)
		runes[i], runes[j] = runes[j], runes[i]
	}
}

func generateBirthday(yearsAgo int, layout string) string {
	now := time.Now()
	birthYear := now.Year() - yearsAgo
	month := time.Month(rand.Intn(12) + 1)
	day := rand.Intn(daysIn(month, birthYear)) + 1
	birthDate := time.Date(birthYear, month, day, 0, 0, 0, 0, time.UTC)
	return birthDate.Format(layout)
}

func daysIn(month time.Month, year int) int {
	if month == time.February {
		if (year%4 == 0 && year%100 != 0) || (year%400 == 0) {
			return 29
		}
		return 28
	}
	if month == time.January || month == time.March || month == time.May ||
		month == time.July || month == time.August || month == time.October || month == time.December {
		return 31
	}
	return 30
}

func generateIban() string {
	part1 := strings.ToUpper(randomLettersString(2))
	part2 := randomNumericString(2)
	part3 := strings.ToUpper(randomLettersString(4))
	part4 := randomNumericString(12)
	return part1 + part2 + part3 + part4
}

func generatePersonalId() string {
	return randomNumericString(6) + "-" + randomNumericString(5)
}

func generateBrandTitle(length int) string {
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

func generateAlias(length int) string {
	if length > 100 {
		length = 100
	}
	if length <= 0 {
		return ""
	}

	allowed := "abcdefghijklmnopqrstuvwxyz0123456789-"
	var sb strings.Builder
	var last byte

	ch := allowed[rand.Intn(len(allowed))]
	sb.WriteByte(ch)
	last = ch

	for i := 1; i < length; i++ {
		if last == '-' {
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

func generateRandomString(length int, charSet string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charSet[rand.Intn(len(charSet))]
	}
	return string(b)
}

func generateCategoryTitle(length int) string {
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
