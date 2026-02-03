package task

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/raksul/go-clickup/clickup"
)

// resolveFieldByName finds a custom field by name (case-insensitive) in a list
// of custom fields.
func resolveFieldByName(fields []clickup.CustomField, name string) *clickup.CustomField {
	for i := range fields {
		if strings.EqualFold(fields[i].Name, name) {
			return &fields[i]
		}
	}
	return nil
}

// parseFieldValue parses a string value into the appropriate type for a
// custom field, returning a value suitable for SetCustomFieldValue.
func parseFieldValue(field *clickup.CustomField, rawValue string) (interface{}, error) {
	switch field.Type {
	case "url", "email", "phone", "text", "short_text":
		return rawValue, nil

	case "number", "currency", "manual_progress":
		f, err := strconv.ParseFloat(rawValue, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid number %q for field %q: %w", rawValue, field.Name, err)
		}
		return f, nil

	case "date":
		return parseDateFieldValue(rawValue)

	case "checkbox":
		v := strings.ToLower(rawValue)
		switch v {
		case "true", "yes", "1":
			return true, nil
		case "false", "no", "0":
			return false, nil
		default:
			return nil, fmt.Errorf("invalid checkbox value %q for field %q (use true/false)", rawValue, field.Name)
		}

	case "dropdown":
		return resolveDropdownOption(field, rawValue)

	case "labels":
		return resolveLabelOptions(field, rawValue)

	case "emoji":
		v, err := strconv.Atoi(rawValue)
		if err != nil {
			return nil, fmt.Errorf("invalid emoji rating %q for field %q: %w", rawValue, field.Name, err)
		}
		return v, nil

	case "users":
		// Expect comma-separated user IDs.
		ids := strings.Split(rawValue, ",")
		var users []map[string]interface{}
		for _, id := range ids {
			id = strings.TrimSpace(id)
			if id != "" {
				users = append(users, map[string]interface{}{"id": id})
			}
		}
		return users, nil

	case "tasks":
		// Expect comma-separated task IDs.
		ids := strings.Split(rawValue, ",")
		var tasks []map[string]interface{}
		for _, id := range ids {
			id = strings.TrimSpace(id)
			if id != "" {
				tasks = append(tasks, map[string]interface{}{"id": id})
			}
		}
		return map[string]interface{}{"add": tasks}, nil

	case "location":
		return parseLocationValue(rawValue, field.Name)

	default:
		return rawValue, nil
	}
}

// parseDateFieldValue parses a date string into unix milliseconds.
func parseDateFieldValue(rawValue string) (interface{}, error) {
	// Try YYYY-MM-DD HH:MM first, then YYYY-MM-DD.
	for _, layout := range []string{"2006-01-02 15:04", "2006-01-02"} {
		if t, err := time.Parse(layout, rawValue); err == nil {
			return t.UnixMilli(), nil
		}
	}
	return nil, fmt.Errorf("invalid date %q (use YYYY-MM-DD or YYYY-MM-DD HH:MM)", rawValue)
}

// resolveDropdownOption matches an option name to its UUID in the field's type_config.
func resolveDropdownOption(field *clickup.CustomField, optionName string) (interface{}, error) {
	options := extractTypeConfigOptions(field.TypeConfig)
	if options == nil {
		return nil, fmt.Errorf("field %q has no dropdown options configured", field.Name)
	}

	for _, opt := range options {
		if name, ok := opt["name"].(string); ok {
			if strings.EqualFold(name, optionName) {
				if id, ok := opt["id"].(string); ok {
					return id, nil
				}
				if orderIdx, ok := opt["orderindex"].(float64); ok {
					return int(orderIdx), nil
				}
			}
		}
	}

	available := listOptionNames(options)
	return nil, fmt.Errorf("option %q not found for field %q (available: %s)", optionName, field.Name, available)
}

// resolveLabelOptions matches comma-separated label names to their UUIDs.
func resolveLabelOptions(field *clickup.CustomField, rawValue string) (interface{}, error) {
	options := extractTypeConfigOptions(field.TypeConfig)
	if options == nil {
		return nil, fmt.Errorf("field %q has no label options configured", field.Name)
	}

	names := strings.Split(rawValue, ",")
	var ids []string
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		found := false
		for _, opt := range options {
			if optName, ok := opt["name"].(string); ok {
				if strings.EqualFold(optName, name) {
					if id, ok := opt["id"].(string); ok {
						ids = append(ids, id)
						found = true
						break
					}
				}
			}
		}
		if !found {
			available := listOptionNames(options)
			return nil, fmt.Errorf("label %q not found for field %q (available: %s)", name, field.Name, available)
		}
	}

	return ids, nil
}

// parseLocationValue parses a "lat,lng,address" or just "address" format.
func parseLocationValue(rawValue, fieldName string) (interface{}, error) {
	parts := strings.SplitN(rawValue, ",", 3)
	if len(parts) >= 3 {
		lat, latErr := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		lng, lngErr := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if latErr == nil && lngErr == nil {
			return map[string]interface{}{
				"location": map[string]interface{}{
					"lat":              lat,
					"lng":              lng,
					"formatted_address": strings.TrimSpace(parts[2]),
				},
			}, nil
		}
	}
	// Treat the whole value as an address.
	return map[string]interface{}{
		"location": map[string]interface{}{
			"formatted_address": rawValue,
		},
	}, nil
}

// extractTypeConfigOptions extracts the "options" array from a field's TypeConfig.
func extractTypeConfigOptions(typeConfig interface{}) []map[string]interface{} {
	tc, ok := typeConfig.(map[string]interface{})
	if !ok {
		return nil
	}
	opts, ok := tc["options"].([]interface{})
	if !ok {
		return nil
	}
	var result []map[string]interface{}
	for _, o := range opts {
		if m, ok := o.(map[string]interface{}); ok {
			result = append(result, m)
		}
	}
	return result
}

// listOptionNames returns a comma-separated list of option names.
func listOptionNames(options []map[string]interface{}) string {
	var names []string
	for _, opt := range options {
		if name, ok := opt["name"].(string); ok {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return "(none)"
	}
	return strings.Join(names, ", ")
}

// formatCustomFieldValue formats a custom field's value for display.
func formatCustomFieldValue(field clickup.CustomField) string {
	if field.Value == nil {
		return ""
	}

	switch field.Type {
	case "url", "email", "phone", "text", "short_text":
		if s, ok := field.Value.(string); ok && s != "" {
			return s
		}

	case "number", "currency":
		if f, ok := field.Value.(float64); ok {
			if field.Type == "currency" {
				prefix := currencyPrefix(field.TypeConfig)
				return fmt.Sprintf("%s%.2f", prefix, f)
			}
			// Format without trailing zeros.
			return strconv.FormatFloat(f, 'f', -1, 64)
		}

	case "date":
		return formatDateFieldValue(field.Value)

	case "checkbox":
		if b, ok := field.Value.(bool); ok {
			if b {
				return "Yes"
			}
			return "No"
		}
		// ClickUp sometimes sends "true"/"false" as strings.
		if s, ok := field.Value.(string); ok {
			switch s {
			case "true":
				return "Yes"
			case "false":
				return "No"
			}
		}

	case "dropdown":
		return formatDropdownValue(field)

	case "labels":
		return formatLabelsValue(field)

	case "users":
		return formatUsersValue(field.Value)

	case "tasks":
		return formatTasksValue(field.Value)

	case "emoji":
		if f, ok := field.Value.(float64); ok {
			return fmt.Sprintf("%d", int(f))
		}

	case "manual_progress", "automatic_progress":
		if m, ok := field.Value.(map[string]interface{}); ok {
			if pct, ok := m["percent_completed"].(float64); ok {
				return fmt.Sprintf("%.0f%%", pct)
			}
			if current, ok := m["current"].(float64); ok {
				return fmt.Sprintf("%.0f%%", current)
			}
		}
		if f, ok := field.Value.(float64); ok {
			return fmt.Sprintf("%.0f%%", f)
		}

	case "location":
		return formatLocationValue(field.Value)

	case "formula":
		if f, ok := field.Value.(float64); ok {
			return strconv.FormatFloat(f, 'f', -1, 64)
		}
		if s, ok := field.Value.(string); ok {
			return s
		}
	}

	// Fallback: try string conversion.
	if s, ok := field.Value.(string); ok && s != "" {
		return s
	}
	return ""
}

// formatDateFieldValue formats a date field value (unix millis as float64 or string).
func formatDateFieldValue(value interface{}) string {
	var ms int64
	switch v := value.(type) {
	case float64:
		ms = int64(v)
	case string:
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return v
		}
		ms = parsed
	default:
		return ""
	}
	t := time.Unix(0, ms*int64(time.Millisecond))
	return t.Format("2006-01-02")
}

// formatDropdownValue looks up the selected option name by matching the value
// to the type_config options.
func formatDropdownValue(field clickup.CustomField) string {
	optionIdx, ok := field.Value.(float64)
	if !ok {
		if s, ok := field.Value.(string); ok {
			// Could be a UUID - try matching.
			options := extractTypeConfigOptions(field.TypeConfig)
			for _, opt := range options {
				if id, ok := opt["id"].(string); ok && id == s {
					if name, ok := opt["name"].(string); ok {
						return name
					}
				}
			}
			return s
		}
		return ""
	}

	options := extractTypeConfigOptions(field.TypeConfig)
	idx := int(optionIdx)
	for _, opt := range options {
		if orderIdx, ok := opt["orderindex"].(float64); ok && int(orderIdx) == idx {
			if name, ok := opt["name"].(string); ok {
				return name
			}
		}
	}
	return fmt.Sprintf("%d", idx)
}

// formatLabelsValue formats a labels field value.
func formatLabelsValue(field clickup.CustomField) string {
	vals, ok := field.Value.([]interface{})
	if !ok {
		return ""
	}

	options := extractTypeConfigOptions(field.TypeConfig)
	optionMap := make(map[string]string)
	for _, opt := range options {
		if id, ok := opt["id"].(string); ok {
			if name, ok := opt["name"].(string); ok {
				optionMap[id] = name
			}
		}
	}

	var names []string
	for _, v := range vals {
		switch val := v.(type) {
		case string:
			if name, ok := optionMap[val]; ok {
				names = append(names, name)
			} else {
				names = append(names, val)
			}
		case float64:
			idx := int(val)
			for _, opt := range options {
				if orderIdx, ok := opt["orderindex"].(float64); ok && int(orderIdx) == idx {
					if name, ok := opt["name"].(string); ok {
						names = append(names, name)
					}
				}
			}
		}
	}

	if len(names) == 0 {
		return ""
	}
	return strings.Join(names, ", ")
}

// formatUsersValue formats a users field value.
func formatUsersValue(value interface{}) string {
	users, ok := value.([]interface{})
	if !ok {
		return ""
	}
	var names []string
	for _, u := range users {
		if m, ok := u.(map[string]interface{}); ok {
			if username, ok := m["username"].(string); ok {
				names = append(names, username)
			} else if email, ok := m["email"].(string); ok {
				names = append(names, email)
			}
		}
	}
	if len(names) == 0 {
		return ""
	}
	return strings.Join(names, ", ")
}

// formatTasksValue formats a tasks field value.
func formatTasksValue(value interface{}) string {
	tasks, ok := value.([]interface{})
	if !ok {
		return ""
	}
	var ids []string
	for _, t := range tasks {
		if m, ok := t.(map[string]interface{}); ok {
			if id, ok := m["id"].(string); ok {
				ids = append(ids, id)
			}
		}
	}
	if len(ids) == 0 {
		return ""
	}
	return strings.Join(ids, ", ")
}

// formatLocationValue formats a location field value.
func formatLocationValue(value interface{}) string {
	m, ok := value.(map[string]interface{})
	if !ok {
		return ""
	}
	loc, ok := m["location"].(map[string]interface{})
	if !ok {
		loc = m // Try the value directly.
	}
	if addr, ok := loc["formatted_address"].(string); ok && addr != "" {
		return addr
	}
	lat, hasLat := loc["lat"].(float64)
	lng, hasLng := loc["lng"].(float64)
	if hasLat && hasLng {
		return fmt.Sprintf("%.6f, %.6f", lat, lng)
	}
	return ""
}

// currencyPrefix extracts the currency symbol from type_config.
func currencyPrefix(typeConfig interface{}) string {
	tc, ok := typeConfig.(map[string]interface{})
	if !ok {
		return "$"
	}
	if sym, ok := tc["currency_type"].(string); ok {
		switch sym {
		case "USD":
			return "$"
		case "EUR":
			return "EUR "
		case "GBP":
			return "GBP "
		case "NZD":
			return "NZ$"
		case "AUD":
			return "A$"
		default:
			return sym + " "
		}
	}
	return "$"
}

// customFieldNames returns a comma-separated list of field names from a task's custom fields.
func customFieldNames(fields []clickup.CustomField) string {
	if len(fields) == 0 {
		return "(none)"
	}
	names := make([]string, 0, len(fields))
	for _, f := range fields {
		names = append(names, f.Name)
	}
	return strings.Join(names, ", ")
}
