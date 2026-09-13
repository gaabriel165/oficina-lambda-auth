package cpf_test

import (
	"testing"

	"github.com/gabrielcamargo/oficina-lambda-auth/internal/cpf"
	"github.com/stretchr/testify/assert"
)

func TestValidate_ShouldAcceptMaskedCPF(t *testing.T) {
	digits, err := cpf.Validate("529.982.247-25")

	assert.NoError(t, err)
	assert.Equal(t, "52998224725", digits)
}

func TestValidate_ShouldAcceptDigitsOnlyCPF(t *testing.T) {
	digits, err := cpf.Validate("98765432100")

	assert.NoError(t, err)
	assert.Equal(t, "98765432100", digits)
}

func TestValidate_ShouldRejectWrongCheckDigit(t *testing.T) {
	_, err := cpf.Validate("52998224726")

	assert.ErrorIs(t, err, cpf.ErrInvalidCPF)
}

func TestValidate_ShouldRejectRepeatedDigits(t *testing.T) {
	_, err := cpf.Validate("11111111111")

	assert.ErrorIs(t, err, cpf.ErrInvalidCPF)
}

func TestValidate_ShouldRejectWrongLength(t *testing.T) {
	_, err := cpf.Validate("1234567890")

	assert.ErrorIs(t, err, cpf.ErrInvalidCPF)
}

func TestMask_ShouldHideMiddleDigits(t *testing.T) {
	assert.Equal(t, "529******25", cpf.Mask("52998224725"))
}
