package task

import (
	"testing"

	"github.com/raksul/go-clickup/clickup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeDropdownField(name string, options []map[string]interface{}) *clickup.CustomField {
	return &clickup.CustomField{
		Name: name,
		Type: "dropdown",
		TypeConfig: map[string]interface{}{
			"options": toInterfaceSlice(options),
		},
	}
}

func makeLabelsField(name string, options []map[string]interface{}) *clickup.CustomField {
	return &clickup.CustomField{
		Name: name,
		Type: "labels",
		TypeConfig: map[string]interface{}{
			"options": toInterfaceSlice(options),
		},
	}
}

func toInterfaceSlice(opts []map[string]interface{}) []interface{} {
	result := make([]interface{}, len(opts))
	for i, o := range opts {
		result[i] = o
	}
	return result
}

// ---------------------------------------------------------------------------
// parseFieldValue
// ---------------------------------------------------------------------------

func TestParseFieldValue(t *testing.T) {
	tests := []struct {
		name      string
		field     *clickup.CustomField
		rawValue  string
		want      interface{}
		wantErr   bool
		errSubstr string
	}{
		{
			name:     "url passthrough",
			field:    &clickup.CustomField{Name: "Website", Type: "url"},
			rawValue: "https://example.com",
			want:     "https://example.com",
		},
		{
			name:     "text passthrough",
			field:    &clickup.CustomField{Name: "Note", Type: "text"},
			rawValue: "hello world",
			want:     "hello world",
		},
		{
			name:     "email passthrough",
			field:    &clickup.CustomField{Name: "Email", Type: "email"},
			rawValue: "a@b.com",
			want:     "a@b.com",
		},
		{
			name:     "number valid",
			field:    &clickup.CustomField{Name: "Count", Type: "number"},
			rawValue: "42.5",
			want:     42.5,
		},
		{
			name:      "number invalid",
			field:     &clickup.CustomField{Name: "Count", Type: "number"},
			rawValue:  "abc",
			wantErr:   true,
			errSubstr: "invalid number",
		},
		{
			name:     "currency valid",
			field:    &clickup.CustomField{Name: "Price", Type: "currency"},
			rawValue: "19.99",
			want:     19.99,
		},
		{
			name:     "date YYYY-MM-DD",
			field:    &clickup.CustomField{Name: "Due", Type: "date"},
			rawValue: "2024-06-15",
		},
		{
			name:     "date YYYY-MM-DD HH:MM",
			field:    &clickup.CustomField{Name: "Due", Type: "date"},
			rawValue: "2024-06-15 14:30",
		},
		{
			name:     "checkbox true",
			field:    &clickup.CustomField{Name: "Done", Type: "checkbox"},
			rawValue: "true",
			want:     true,
		},
		{
			name:     "checkbox yes",
			field:    &clickup.CustomField{Name: "Done", Type: "checkbox"},
			rawValue: "yes",
			want:     true,
		},
		{
			name:     "checkbox 1",
			field:    &clickup.CustomField{Name: "Done", Type: "checkbox"},
			rawValue: "1",
			want:     true,
		},
		{
			name:     "checkbox false",
			field:    &clickup.CustomField{Name: "Done", Type: "checkbox"},
			rawValue: "false",
			want:     false,
		},
		{
			name:     "checkbox no",
			field:    &clickup.CustomField{Name: "Done", Type: "checkbox"},
			rawValue: "no",
			want:     false,
		},
		{
			name:     "checkbox 0",
			field:    &clickup.CustomField{Name: "Done", Type: "checkbox"},
			rawValue: "0",
			want:     false,
		},
		{
			name:      "checkbox invalid",
			field:     &clickup.CustomField{Name: "Done", Type: "checkbox"},
			rawValue:  "maybe",
			wantErr:   true,
			errSubstr: "invalid checkbox value",
		},
		{
			name: "dropdown match",
			field: makeDropdownField("Priority", []map[string]interface{}{
				{"id": "uuid1", "name": "High", "orderindex": float64(0)},
				{"id": "uuid2", "name": "Low", "orderindex": float64(1)},
			}),
			rawValue: "High",
			want:     "uuid1",
		},
		{
			name: "dropdown no match",
			field: makeDropdownField("Priority", []map[string]interface{}{
				{"id": "uuid1", "name": "High", "orderindex": float64(0)},
			}),
			rawValue:  "Medium",
			wantErr:   true,
			errSubstr: "not found",
		},
		{
			name:      "dropdown nil TypeConfig",
			field:     &clickup.CustomField{Name: "Priority", Type: "dropdown", TypeConfig: nil},
			rawValue:  "High",
			wantErr:   true,
			errSubstr: "no dropdown options",
		},
		{
			name: "labels single",
			field: makeLabelsField("Tags", []map[string]interface{}{
				{"id": "l1", "label": "Bug"},
				{"id": "l2", "label": "Feature"},
			}),
			rawValue: "Bug",
			want:     []string{"l1"},
		},
		{
			name: "labels comma-separated",
			field: makeLabelsField("Tags", []map[string]interface{}{
				{"id": "l1", "label": "Bug"},
				{"id": "l2", "label": "Feature"},
			}),
			rawValue: "Bug, Feature",
			want:     []string{"l1", "l2"},
		},
		{
			name: "labels not found",
			field: makeLabelsField("Tags", []map[string]interface{}{
				{"id": "l1", "label": "Bug"},
			}),
			rawValue:  "Missing",
			wantErr:   true,
			errSubstr: "not found",
		},
		{
			name:     "emoji valid",
			field:    &clickup.CustomField{Name: "Rating", Type: "emoji"},
			rawValue: "3",
			want:     3,
		},
		{
			name:      "emoji invalid",
			field:     &clickup.CustomField{Name: "Rating", Type: "emoji"},
			rawValue:  "abc",
			wantErr:   true,
			errSubstr: "invalid emoji rating",
		},
		{
			name:     "users comma-separated",
			field:    &clickup.CustomField{Name: "Assignees", Type: "users"},
			rawValue: "123, 456",
			want: []map[string]interface{}{
				{"id": "123"},
				{"id": "456"},
			},
		},
		{
			name:     "tasks comma-separated",
			field:    &clickup.CustomField{Name: "Related", Type: "tasks"},
			rawValue: "abc, def",
			want: map[string]interface{}{
				"add": []map[string]interface{}{
					{"id": "abc"},
					{"id": "def"},
				},
			},
		},
		{
			name:     "location lat,lng,address",
			field:    &clickup.CustomField{Name: "Place", Type: "location"},
			rawValue: "40.7128,-74.0060,New York",
			want: map[string]interface{}{
				"location": map[string]interface{}{
					"lat":               40.7128,
					"lng":               -74.0060,
					"formatted_address": "New York",
				},
			},
		},
		{
			name:     "location address only",
			field:    &clickup.CustomField{Name: "Place", Type: "location"},
			rawValue: "123 Main Street",
			want: map[string]interface{}{
				"location": map[string]interface{}{
					"formatted_address": "123 Main Street",
				},
			},
		},
		{
			name:     "location invalid lat with commas",
			field:    &clickup.CustomField{Name: "Place", Type: "location"},
			rawValue: "notanum,alsonotanum,Some Place",
			want: map[string]interface{}{
				"location": map[string]interface{}{
					"formatted_address": "notanum,alsonotanum,Some Place",
				},
			},
		},
		{
			name:     "unknown type passthrough",
			field:    &clickup.CustomField{Name: "Mystery", Type: "some_future_type"},
			rawValue: "whatever",
			want:     "whatever",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseFieldValue(tt.field, tt.rawValue)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
				return
			}
			require.NoError(t, err)
			if tt.want != nil {
				assert.Equal(t, tt.want, got)
			} else {
				// For date type, just check it's non-nil (millis depend on timezone).
				assert.NotNil(t, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// parseDateFieldValue
// ---------------------------------------------------------------------------

func TestParseDateFieldValue(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid date", "2024-01-15", false},
		{"valid datetime", "2024-01-15 09:30", false},
		{"invalid", "not-a-date", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDateFieldValue(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.IsType(t, int64(0), got)
		})
	}
}

// ---------------------------------------------------------------------------
// resolveDropdownOption
// ---------------------------------------------------------------------------

func TestResolveDropdownOption(t *testing.T) {
	tests := []struct {
		name      string
		field     *clickup.CustomField
		option    string
		want      interface{}
		wantErr   bool
		errSubstr string
	}{
		{
			name: "match by name",
			field: makeDropdownField("Status", []map[string]interface{}{
				{"id": "uuid1", "name": "Open", "orderindex": float64(0)},
			}),
			option: "Open",
			want:   "uuid1",
		},
		{
			name: "case-insensitive match",
			field: makeDropdownField("Status", []map[string]interface{}{
				{"id": "uuid1", "name": "Open", "orderindex": float64(0)},
			}),
			option: "open",
			want:   "uuid1",
		},
		{
			name: "no match",
			field: makeDropdownField("Status", []map[string]interface{}{
				{"id": "uuid1", "name": "Open", "orderindex": float64(0)},
			}),
			option:    "Closed",
			wantErr:   true,
			errSubstr: "not found",
		},
		{
			name:      "nil TypeConfig",
			field:     &clickup.CustomField{Name: "Status", Type: "dropdown", TypeConfig: nil},
			option:    "Open",
			wantErr:   true,
			errSubstr: "no dropdown options",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveDropdownOption(tt.field, tt.option)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errSubstr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// resolveLabelOptions
// ---------------------------------------------------------------------------

func TestResolveLabelOptions(t *testing.T) {
	tests := []struct {
		name      string
		field     *clickup.CustomField
		rawValue  string
		want      []string
		wantErr   bool
		errSubstr string
	}{
		{
			name: "single label",
			field: makeLabelsField("Tags", []map[string]interface{}{
				{"id": "l1", "label": "Bug"},
			}),
			rawValue: "Bug",
			want:     []string{"l1"},
		},
		{
			name: "multiple labels",
			field: makeLabelsField("Tags", []map[string]interface{}{
				{"id": "l1", "label": "Bug"},
				{"id": "l2", "label": "Feature"},
				{"id": "l3", "label": "Urgent"},
			}),
			rawValue: "Bug, Urgent",
			want:     []string{"l1", "l3"},
		},
		{
			name: "one not found",
			field: makeLabelsField("Tags", []map[string]interface{}{
				{"id": "l1", "label": "Bug"},
			}),
			rawValue:  "Bug, Nope",
			wantErr:   true,
			errSubstr: `label "Nope" not found`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveLabelOptions(tt.field, tt.rawValue)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errSubstr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// parseLocationValue
// ---------------------------------------------------------------------------

func TestParseLocationValue(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  map[string]interface{}
	}{
		{
			name:  "lat,lng,address",
			input: "-36.8485,174.7633,Auckland NZ",
			want: map[string]interface{}{
				"location": map[string]interface{}{
					"lat":               -36.8485,
					"lng":               174.7633,
					"formatted_address": "Auckland NZ",
				},
			},
		},
		{
			name:  "address only",
			input: "Wellington, New Zealand",
			want: map[string]interface{}{
				"location": map[string]interface{}{
					"formatted_address": "Wellington, New Zealand",
				},
			},
		},
		{
			name:  "invalid lat with commas falls back to address",
			input: "abc,def,Some Place",
			want: map[string]interface{}{
				"location": map[string]interface{}{
					"formatted_address": "abc,def,Some Place",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseLocationValue(tt.input, "Place")
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// formatCustomFieldValue
// ---------------------------------------------------------------------------

func TestFormatCustomFieldValue(t *testing.T) {
	tests := []struct {
		name  string
		field clickup.CustomField
		want  string
	}{
		{
			name:  "nil value",
			field: clickup.CustomField{Type: "text", Value: nil},
			want:  "",
		},
		{
			name:  "text string",
			field: clickup.CustomField{Type: "text", Value: "hello"},
			want:  "hello",
		},
		{
			name:  "url string",
			field: clickup.CustomField{Type: "url", Value: "https://example.com"},
			want:  "https://example.com",
		},
		{
			name:  "number integer-like",
			field: clickup.CustomField{Type: "number", Value: float64(42)},
			want:  "42",
		},
		{
			name:  "number with decimal",
			field: clickup.CustomField{Type: "number", Value: float64(3.14)},
			want:  "3.14",
		},
		{
			name: "currency USD",
			field: clickup.CustomField{
				Type:       "currency",
				Value:      float64(19.99),
				TypeConfig: map[string]interface{}{"currency_type": "USD"},
			},
			want: "$19.99",
		},
		{
			name: "currency EUR",
			field: clickup.CustomField{
				Type:       "currency",
				Value:      float64(10),
				TypeConfig: map[string]interface{}{"currency_type": "EUR"},
			},
			want: "EUR 10.00",
		},
		{
			name:  "checkbox true",
			field: clickup.CustomField{Type: "checkbox", Value: true},
			want:  "Yes",
		},
		{
			name:  "checkbox false",
			field: clickup.CustomField{Type: "checkbox", Value: false},
			want:  "No",
		},
		{
			name:  "checkbox string true",
			field: clickup.CustomField{Type: "checkbox", Value: "true"},
			want:  "Yes",
		},
		{
			name:  "emoji float",
			field: clickup.CustomField{Type: "emoji", Value: float64(5)},
			want:  "5",
		},
		{
			name: "date float64 millis",
			field: clickup.CustomField{
				Type:  "date",
				Value: float64(1718409600000), // 2024-06-15 UTC
			},
			want: "2024-06-15",
		},
		{
			name:  "date string millis",
			field: clickup.CustomField{Type: "date", Value: "1718409600000"},
			want:  "2024-06-15",
		},
		{
			name: "location with address",
			field: clickup.CustomField{
				Type: "location",
				Value: map[string]interface{}{
					"location": map[string]interface{}{
						"formatted_address": "Auckland",
					},
				},
			},
			want: "Auckland",
		},
		{
			name:  "formula float",
			field: clickup.CustomField{Type: "formula", Value: float64(99)},
			want:  "99",
		},
		{
			name:  "formula string",
			field: clickup.CustomField{Type: "formula", Value: "calculated"},
			want:  "calculated",
		},
		{
			name:  "manual_progress float",
			field: clickup.CustomField{Type: "manual_progress", Value: float64(75)},
			want:  "75%",
		},
		{
			name: "manual_progress map with percent",
			field: clickup.CustomField{
				Type:  "manual_progress",
				Value: map[string]interface{}{"percent_completed": float64(50)},
			},
			want: "50%",
		},
		{
			name:  "unknown type with string fallback",
			field: clickup.CustomField{Type: "some_new_type", Value: "raw"},
			want:  "raw",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatCustomFieldValue(tt.field)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// formatDateFieldValue
// ---------------------------------------------------------------------------

func TestFormatDateFieldValue(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  string
	}{
		{"float64 millis", float64(1718409600000), "2024-06-15"},
		{"string millis", "1718409600000", "2024-06-15"},
		{"invalid string", "not-a-number", "not-a-number"},
		{"unsupported type", 42, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDateFieldValue(tt.value)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// resolveFieldByName
// ---------------------------------------------------------------------------

func TestResolveFieldByName(t *testing.T) {
	fields := []clickup.CustomField{
		{Name: "Priority"},
		{Name: "Status"},
		{Name: "Due Date"},
	}

	tests := []struct {
		name     string
		query    string
		wantName string
		wantNil  bool
	}{
		{"exact match", "Priority", "Priority", false},
		{"case-insensitive", "priority", "Priority", false},
		{"not found", "nonexistent", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveFieldByName(fields, tt.query)
			if tt.wantNil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tt.wantName, result.Name)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// customFieldNames
// ---------------------------------------------------------------------------

func TestCustomFieldNames(t *testing.T) {
	tests := []struct {
		name   string
		fields []clickup.CustomField
		want   string
	}{
		{
			name:   "multiple fields",
			fields: []clickup.CustomField{{Name: "A"}, {Name: "B"}, {Name: "C"}},
			want:   "A, B, C",
		},
		{
			name:   "empty",
			fields: nil,
			want:   "(none)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, customFieldNames(tt.fields))
		})
	}
}

// ---------------------------------------------------------------------------
// currencyPrefix
// ---------------------------------------------------------------------------

func TestCurrencyPrefix(t *testing.T) {
	tests := []struct {
		name       string
		typeConfig interface{}
		want       string
	}{
		{"USD", map[string]interface{}{"currency_type": "USD"}, "$"},
		{"EUR", map[string]interface{}{"currency_type": "EUR"}, "EUR "},
		{"NZD", map[string]interface{}{"currency_type": "NZD"}, "NZ$"},
		{"AUD", map[string]interface{}{"currency_type": "AUD"}, "A$"},
		{"unknown currency", map[string]interface{}{"currency_type": "JPY"}, "JPY "},
		{"nil TypeConfig", nil, "$"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, currencyPrefix(tt.typeConfig))
		})
	}
}
