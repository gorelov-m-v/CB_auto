package main

import (
	"fmt"
	"strconv"
)

func maximumSalary(numStr string) string {
	if len(numStr) == 1 {
		return "0"
	}

	for i := 0; i < len(numStr)-1; i++ {
		// Преобразуем символы в числа для сравнения
		currDigit, _ := strconv.Atoi(string(numStr[i]))   // Текущая цифра
		nextDigit, _ := strconv.Atoi(string(numStr[i+1])) // Следующая цифра

		if currDigit > nextDigit {
			continue
		}

		if currDigit < nextDigit {
			// Удаляем цифру с индексом i
			newStr := numStr[:i] + numStr[i+1:]
			return newStr
		}
	}

	// Если ничего не удалили, удаляем последнюю цифру
	return numStr[:len(numStr)-1]
}

func main() {
	var testCount int
	fmt.Scan(&testCount)

	for i := 0; i < testCount; i++ {
		var salaryStr string
		fmt.Scan(&salaryStr)
		result := maximumSalary(salaryStr)
		fmt.Println(result)
	}
}
