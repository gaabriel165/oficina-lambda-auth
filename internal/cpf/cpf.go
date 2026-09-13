package cpf

import (
	"errors"
	"strings"
)

var ErrInvalidCPF = errors.New("invalid cpf")

const cpfLength = 11

func Validate(raw string) (string, error) {
	digits := onlyDigits(raw)
	if len(digits) != cpfLength {
		return "", ErrInvalidCPF
	}
	if allSameDigits(digits) {
		return "", ErrInvalidCPF
	}
	if checkDigit(digits[:9], 10) != int(digits[9]-'0') {
		return "", ErrInvalidCPF
	}
	if checkDigit(digits[:10], 11) != int(digits[10]-'0') {
		return "", ErrInvalidCPF
	}
	return digits, nil
}

func Mask(digits string) string {
	if len(digits) != cpfLength {
		return "***********"
	}
	return digits[:3] + "******" + digits[9:]
}

func onlyDigits(raw string) string {
	var builder strings.Builder
	for _, char := range raw {
		if char >= '0' && char <= '9' {
			builder.WriteRune(char)
		}
	}
	return builder.String()
}

func allSameDigits(digits string) bool {
	return strings.Count(digits, string(digits[0])) == len(digits)
}

func checkDigit(base string, weight int) int {
	sum := 0
	for _, char := range base {
		sum += int(char-'0') * weight
		weight--
	}
	remainder := (sum * 10) % 11
	if remainder == 10 {
		return 0
	}
	return remainder
}
