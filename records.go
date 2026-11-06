package main

import "strings"

// Packed record wire format: recordMarker followed by fieldSep-joined "FIELD=value" chunks.
const (
	recordMarker = "\x1e"
	fieldSep     = "\x1f"
)

func packRecord(parts []string) string { return recordMarker + strings.Join(parts, fieldSep) }

func isPackedRecord(s string) bool { return strings.HasPrefix(s, recordMarker) }

// forEachPackedField decodes chunks; chunks without "=" are bound positionally to tmplFields.
func forEachPackedField(packed string, tmplFields []string, fn func(field, val string)) {
	clean := strings.TrimPrefix(packed, recordMarker)
	for idx, chunk := range strings.Split(clean, fieldSep) {
		if eq := strings.IndexByte(chunk, '='); eq != -1 {
			fn(chunk[:eq], chunk[eq+1:])
		} else if idx < len(tmplFields) {
			fn(tmplFields[idx], chunk)
		}
	}
}

func unpackRecordIntoMap(m map[string]string, key, val string, tmplFields []string) {
	prefix := key + "."
	for k := range m {
		if strings.HasPrefix(k, prefix) {
			delete(m, k)
		}
	}
	if strings.TrimPrefix(val, recordMarker) == "" {
		return
	}
	forEachPackedField(val, tmplFields, func(field, v string) {
		m[prefix+field] = coerceBySigil(field, v)
	})
}

// collectRecordFields returns "FIELD=value" chunks for every key in m under prefix (e.g. "P@.").
func collectRecordFields(m map[string]string, prefix string) []string {
	var parts []string
	for k, v := range m {
		if strings.HasPrefix(k, prefix) {
			parts = append(parts, strings.TrimPrefix(k, prefix)+"="+v)
		}
	}
	return parts
}

func formatRecordDisplay(s string) string {
	clean := strings.TrimPrefix(s, recordMarker)
	return strings.ReplaceAll(clean, fieldSep, "\t")
}

func (vm *BasicVM) getTemplateFields(varOrTypeName string) ([]string, bool) {
	key := strings.ToUpper(varOrTypeName)
	if tName, ok := vm.varTypes.Load(key); ok {
		key = tName.(string)
	}
	if fieldsAny, ok := vm.templates.Load(key); ok {
		return fieldsAny.([]string), true
	}
	return nil, false
}
