package docker

import "testing"

func TestValidateIntegerInRange(t *testing.T) {
	validIntegers := []int{-259, 0, 1, 5, 999}
	min := -259
	max := 999
	for _, v := range validIntegers {
		_, errors := validateIntegerInRange(min, max)(v, "name")
		if len(errors) != 0 {
			t.Fatalf("%q should be an integer in range (%d, %d): %q", v, min, max, errors)
		}
	}

	invalidIntegers := []int{-260, -99999, 1000, 25678}
	for _, v := range invalidIntegers {
		_, errors := validateIntegerInRange(min, max)(v, "name")
		if len(errors) == 0 {
			t.Fatalf("%q should be an integer outside range (%d, %d)", v, min, max)
		}
	}
}

func TestValidateIntegerGeqThan0(t *testing.T) {
	v := 1
	if _, error := validateIntegerGeqThan(0)(v, "name"); error != nil {
		t.Fatalf("%q should be an integer greater than 0", v)
	}

	v = -4
	if _, error := validateIntegerGeqThan(0)(v, "name"); error == nil {
		t.Fatalf("%q should be an invalid integer smaller than 0", v)
	}
}

func TestValidateFloatRatio(t *testing.T) {
	v := 0.9
	if _, error := validateFloatRatio()(v, "name"); error != nil {
		t.Fatalf("%v should be a float between 0.0 and 1.0", v)
	}

	v = -4.5
	if _, error := validateFloatRatio()(v, "name"); error == nil {
		t.Fatalf("%v should be an invalid float smaller than 0.0", v)
	}

	v = 1.1
	if _, error := validateFloatRatio()(v, "name"); error == nil {
		t.Fatalf("%v should be an invalid float greater than 1.0", v)
	}
}
func TestValidateDurationGeq0(t *testing.T) {
	v := "1ms"
	if _, error := validateDurationGeq0()(v, "name"); error != nil {
		t.Fatalf("%v should be a valid durarion", v)
	}

	v = "-2h"
	if _, error := validateDurationGeq0()(v, "name"); error == nil {
		t.Fatalf("%v should be an invalid duration smaller than 0", v)
	}
}

func TestValidateStringMatchesPattern(t *testing.T) {
	pattern := `^(pause|continue-mate|break)$`
	v := "pause"
	if _, error := validateStringMatchesPattern(pattern)(v, "name"); error != nil {
		t.Fatalf("%q should match the pattern", v)
	}
	v = "doesnotmatch"
	if _, error := validateStringMatchesPattern(pattern)(v, "name"); error == nil {
		t.Fatalf("%q should not match the pattern", v)
	}
	v = "continue-mate"
	if _, error := validateStringMatchesPattern(pattern)(v, "name"); error != nil {
		t.Fatalf("%q should match the pattern", v)
	}
}

func TestValidateStringShouldBeBase64Encoded(t *testing.T) {
	v := `YmtzbGRrc2xka3NkMjM4MQ==`
	if _, error := validateStringIsBase64Encoded()(v, "name"); error != nil {
		t.Fatalf("%q should be base64 decodeable", v)
	}

	v = `%&df#3NkMjM4MQ==`
	if _, error := validateStringIsBase64Encoded()(v, "name"); error == nil {
		t.Fatalf("%q should NOT be base64 decodeable", v)
	}
}
