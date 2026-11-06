package main

import (
	"math"
	"strconv"
	"strings"
)

// All BASIC values are strings; the name's sigil ($ string, % integer, @ record) decides the type.

// formatNum formats a float64 as a string, returning "0" for NaN or Inf.
func formatNum(f float64) string {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return "0"
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// toFloat converts a string to a float64, returning the value and a boolean indicating success.
func toFloat(s string) (float64, bool) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(trimmed, 64)
	return v, err == nil
}
// isTruthy determines if a string represents a truthy value (non-zero number or non-empty string).
func isTruthy(s string) bool {
	if n, ok := toFloat(s); ok {
		return n != 0
	}
	return s != ""
}

// coerceBySigil converts the value to the appropriate type based on the variable's sigil.
func coerceBySigil(name, val string) string {
	if strings.HasSuffix(name, "%") {
		f, _ := toFloat(val)
		return strconv.FormatInt(int64(f), 10)
	}
	return val
}
// defaultBySigil returns the default value for a variable based on its sigil.
func defaultBySigil(name string) string {
	if strings.HasSuffix(name, "$") || strings.HasSuffix(name, "@") {
		return ""
	}
	return "0"
}

// isStringOrRecName checks if the variable name represents a string or record type, or special names like SPC or TAB.
func isStringOrRecName(name string) bool {
	return strings.HasSuffix(name, "$") || strings.HasSuffix(name, "@") || name == "SPC" || name == "TAB"
}

// baseVarName returns the base variable name by stripping any record field suffix: "P@.X%" -> "P@".
func baseVarName(key string) string {
	if dot := strings.IndexByte(key, '.'); dot != -1 {
		return key[:dot]
	}
	return key
}
