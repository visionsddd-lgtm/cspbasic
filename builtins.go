package main

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// callBuiltin evaluates an intrinsic function; ok is false when name is not a builtin.
func (ctx *execContext) callBuiltin(name string, args []string) (result string, ok bool) {
	vm := ctx.vm
	arg0 := ""
	if len(args) > 0 {
		arg0 = args[0]
	}
	num0, _ := toFloat(arg0)

	switch name {
	// Console and machine
	case "INKEY$":
		return pollInkey(), true
	case "CSRLIN":
		return strconv.Itoa(int(vm.cursor.row.Load())), true
	case "POS":
		return strconv.Itoa(int(vm.cursor.col.Load())), true
	case "PEEK":
		return strconv.Itoa(int(vm.mem.peek(int(num0)))), true
	case "TAB":
		col := int(vm.cursor.col.Load())
		if int(num0) > col {
			return strings.Repeat(" ", int(num0)-col), true
		}
		return "", true
	case "TIMER":
		return fmt.Sprintf("%.3f", time.Since(vm.startTime).Seconds()), true

	// Strings
	case "SPC", "SPACE$":
		return strings.Repeat(" ", max(0, int(num0))), true
	case "LEN":
		return strconv.Itoa(len([]rune(arg0))), true
	case "LEFT$":
		if len(args) < 2 {
			return arg0, true
		}
		n, _ := toFloat(args[1])
		runes := []rune(arg0)
		return string(runes[:max(0, min(int(n), len(runes)))]), true
	case "RIGHT$":
		if len(args) < 2 {
			return arg0, true
		}
		n, _ := toFloat(args[1])
		runes := []rune(arg0)
		k := max(0, min(int(n), len(runes)))
		return string(runes[len(runes)-k:]), true
	case "MID$":
		if len(args) < 2 {
			return "", true
		}
		runes := []rune(arg0)
		startF, _ := toFloat(args[1])
		start := max(0, int(startF)-1)
		if start >= len(runes) {
			return "", true
		}
		if len(args) >= 3 {
			lenF, _ := toFloat(args[2])
			if int(lenF) <= 0 {
				return "", true
			}
			return string(runes[start:min(start+int(lenF), len(runes))]), true
		}
		return string(runes[start:]), true
	case "INSTR":
		if len(args) == 2 {
			idx := strings.Index(args[0], args[1])
			if idx == -1 {
				return "0", true
			}
			return strconv.Itoa(len([]rune(args[0][:idx])) + 1), true
		}
		if len(args) >= 3 {
			start := int(num0) - 1
			runes := []rune(args[1])
			if start < 0 || start >= len(runes) {
				return "0", true
			}
			sub := string(runes[start:])
			idx := strings.Index(sub, args[2])
			if idx == -1 {
				return "0", true
			}
			return strconv.Itoa(start + len([]rune(sub[:idx])) + 1), true
		}
		return "0", true
	case "UCASE$":
		return strings.ToUpper(arg0), true
	case "LCASE$":
		return strings.ToLower(arg0), true
	case "TRIM$":
		return strings.TrimSpace(arg0), true
	case "CHR$":
		return string(rune(int(num0))), true
	case "ASC":
		if len(arg0) == 0 {
			return "0", true
		}
		return strconv.Itoa(int(arg0[0])), true
	case "STR$":
		return formatNum(num0), true
	case "VAL":
		s := strings.TrimSpace(arg0)
		var numStr strings.Builder
		for i, r := range s {
			if unicode.IsDigit(r) || r == '.' || (i == 0 && (r == '-' || r == '+')) {
				numStr.WriteRune(r)
			} else {
				break
			}
		}
		v, _ := toFloat(numStr.String())
		return formatNum(v), true
	case "STRING$":
		if len(args) < 2 || int(num0) <= 0 {
			return "", true
		}
		ch := args[1]
		if cNum, ok := toFloat(ch); ok && len(ch) > 1 {
			ch = string(rune(int(cNum)))
		} else if len([]rune(ch)) > 0 {
			ch = string([]rune(ch)[0])
		}
		return strings.Repeat(ch, int(num0)), true
	case "HEX$":
		return fmt.Sprintf("%X", int64(num0)), true
	case "OCT$":
		return fmt.Sprintf("%o", int64(num0)), true

	// Math
	case "ABS":
		return formatNum(math.Abs(num0)), true
	case "INT":
		return formatNum(math.Floor(num0)), true
	case "FIX":
		return formatNum(math.Trunc(num0)), true
	case "SGN":
		if num0 > 0 {
			return "1", true
		} else if num0 < 0 {
			return "-1", true
		}
		return "0", true
	case "SQR":
		return formatNum(math.Sqrt(max(0, num0))), true
	case "SIN":
		return formatNum(math.Sin(num0)), true
	case "COS":
		return formatNum(math.Cos(num0)), true
	case "TAN":
		return formatNum(math.Tan(num0)), true
	case "ATN":
		return formatNum(math.Atan(num0)), true
	case "LOG":
		if num0 <= 0 {
			return "0", true
		}
		return formatNum(math.Log(num0)), true
	case "EXP":
		return formatNum(math.Exp(num0)), true
	case "MIN":
		if len(args) < 2 {
			return formatNum(num0), true
		}
		num1, _ := toFloat(args[1])
		return formatNum(math.Min(num0, num1)), true
	case "MAX":
		if len(args) < 2 {
			return formatNum(num0), true
		}
		num1, _ := toFloat(args[1])
		return formatNum(math.Max(num0, num1)), true
	case "RND":
		return formatNum(rand.Float64()), true
	case "RANDINT":
		if len(args) >= 2 {
			minV := int(num0)
			maxF, _ := toFloat(args[1])
			if int(maxF) > minV {
				return strconv.Itoa(minV + rand.Intn(int(maxF)-minV+1)), true
			}
			return strconv.Itoa(minV), true
		}
		return "0", true

	// Arrays, DATA and channels
	case "UBOUND":
		dim := 1
		if len(args) >= 2 {
			d, _ := toFloat(args[1])
			dim = int(d)
		}
		return strconv.Itoa(vm.arrays.LoadOrAutoDim(arg0, dim).UBound(dim)), true
	case "DREM":
		return strconv.Itoa(vm.data.remaining()), true
	case "QLEN", "LOF":
		if bc, exists := vm.chans.Load(arg0); exists {
			return strconv.Itoa(len(bc.ch)), true
		}
		return "0", true
	case "EOF":
		if arg0 == "" {
			return ctx.getVar("EOF$"), true
		}
		if bc, exists := vm.chans.Load(arg0); exists {
			if bc.IsEOF() {
				return "1", true
			}
			return "0", true
		}
		return "1", true
	}
	return "", false
}
