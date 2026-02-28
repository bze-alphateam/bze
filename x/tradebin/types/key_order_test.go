package types

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// transformPriceOld is the original implementation using float64
// Kept here for comparison testing to ensure new implementation matches
func transformPriceOld(price string) string {
	floatVal, err := strconv.ParseFloat(price, 64)
	if err != nil {
		return price
	}

	// Format the float back into a string with zero padding to ensure it's 24 characters long
	// Adjust the precision as needed
	return fmt.Sprintf("%024.10f", floatVal)
}

func TestTransformPrice(t *testing.T) {
	testCases := []struct {
		name           string
		price          string
		expectedResult string // Expected output with full 18 decimal precision
	}{
		{
			name:           "simple integer price",
			price:          "100",
			expectedResult: "0000000000100.000000000000000000",
		},
		{
			name:           "price with decimals",
			price:          "123.456",
			expectedResult: "0000000000123.456000000000000000",
		},
		{
			name:           "very small price",
			price:          "0.00001",
			expectedResult: "0000000000000.000010000000000000",
		},
		{
			name:           "very large price",
			price:          "999999999.123456789",
			expectedResult: "0000999999999.123456789000000000", // Full precision preserved
		},
		{
			name:           "price with many decimals",
			price:          "1.123456789012345",
			expectedResult: "0000000000001.123456789012345000",
		},
		{
			name:           "zero price",
			price:          "0",
			expectedResult: "0000000000000.000000000000000000",
		},
		{
			name:           "price with leading zeros",
			price:          "00123.45",
			expectedResult: "0000000000123.450000000000000000",
		},
		{
			name:           "price with trailing zeros",
			price:          "123.4500000",
			expectedResult: "0000000000123.450000000000000000",
		},
		{
			name:           "price just below 1",
			price:          "0.999999999",
			expectedResult: "0000000000000.999999999000000000",
		},
		{
			name:           "price just above 1",
			price:          "1.000000001",
			expectedResult: "0000000000001.000000001000000000",
		},
		{
			name:           "typical crypto price",
			price:          "0.000123456",
			expectedResult: "0000000000000.000123456000000000",
		},
		{
			name:           "typical stock price",
			price:          "145.67",
			expectedResult: "0000000000145.670000000000000000",
		},
		{
			name:           "max precision price with 18 decimals",
			price:          "12345.123456789012345678",
			expectedResult: "0000000012345.123456789012345678",
		},
		{
			name:           "invalid price - should return as is",
			price:          "invalid",
			expectedResult: "invalid",
		},
		{
			name:           "empty string",
			price:          "",
			expectedResult: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			newResult := transformPrice(tc.price)

			// The new implementation should produce the expected result with improved precision
			require.Equal(t, tc.expectedResult, newResult,
				"transformPrice output incorrect for price: %s\nExpected: %s\nGot: %s",
				tc.price, tc.expectedResult, newResult)
		})
	}
}

func TestTransformPriceComparisonWithOld(t *testing.T) {
	// This test documents the differences between old (float64, 10 decimals)
	// and new (LegacyDec, 18 decimals) implementations
	testCases := []struct {
		name        string
		price       string
		description string
	}{
		{
			name:        "simple price - different format",
			price:       "123.456",
			description: "Old: 24 chars/10 decimals, New: 32 chars/18 decimals",
		},
		{
			name:        "large price - improved precision",
			price:       "999999999.123456789",
			description: "Old had float64 rounding bugs, new preserves full precision",
		},
		{
			name:        "large int price - no float64 corruption",
			price:       "123456789.12",
			description: "Old: float64 corruption, New: exact precision",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			oldResult := transformPriceOld(tc.price)
			newResult := transformPrice(tc.price)

			// Formats are intentionally different (24 vs 32 chars, 10 vs 18 decimals)
			require.NotEqual(t, oldResult, newResult,
				"Old and new formats differ for price: %s\nOld (24 chars, 10 decimals): %s\nNew (32 chars, 18 decimals): %s\nReason: %s",
				tc.price, oldResult, newResult, tc.description)

			// Verify new format is correct
			require.Equal(t, 32, len(newResult), "New format should be 32 characters")
			require.Equal(t, 24, len(oldResult), "Old format was 24 characters")
		})
	}
}

func TestTransformPriceFormat(t *testing.T) {
	testCases := []struct {
		name           string
		price          string
		expectedLength int
		shouldBeValid  bool
	}{
		{
			name:           "normal price should be 32 chars",
			price:          "123.456",
			expectedLength: 32,
			shouldBeValid:  true,
		},
		{
			name:           "zero should be 32 chars",
			price:          "0",
			expectedLength: 32,
			shouldBeValid:  true,
		},
		{
			name:           "invalid price returns as is",
			price:          "invalid",
			expectedLength: 7,
			shouldBeValid:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := transformPrice(tc.price)
			require.Equal(t, tc.expectedLength, len(result),
				"Expected length %d but got %d for price: %s (result: %s)",
				tc.expectedLength, len(result), tc.price, result)

			if tc.shouldBeValid {
				// Verify format: 13 digits + dot + 18 decimals
				require.Contains(t, result, ".")
				parts := strings.Split(result, ".")
				require.Len(t, parts, 2)
				require.Equal(t, 13, len(parts[0]), "Integer part should be 13 digits")
				require.Equal(t, 18, len(parts[1]), "Fractional part should be 18 digits")
			}
		})
	}
}

func TestTransformPricePadding(t *testing.T) {
	testCases := []struct {
		name           string
		price          string
		expectedResult string
	}{
		{
			name:           "small number needs padding",
			price:          "1.5",
			expectedResult: "0000000000001.500000000000000000",
		},
		{
			name:           "large number with full precision - no float64 corruption",
			price:          "123456789.12",
			expectedResult: "0000123456789.120000000000000000", // Full 18 decimal precision
		},
		{
			name:           "very high precision price",
			price:          "99.123456789012345678",
			expectedResult: "0000000000099.123456789012345678",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := transformPrice(tc.price)

			// Verify correct 32-character result with full precision
			require.Equal(t, 32, len(result), "Result should be 32 characters")
			require.Equal(t, tc.expectedResult, result)

			// Verify padding is with zeros
			require.True(t, result[0] == '0', "Should start with zero padding")
		})
	}
}
