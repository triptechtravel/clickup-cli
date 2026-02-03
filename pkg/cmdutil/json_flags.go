package cmdutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"text/template"

	"github.com/itchyny/gojq"
	"github.com/spf13/cobra"
)

// JSONFlags holds the --json, --jq, and --template flags.
type JSONFlags struct {
	JSON     bool
	JQ       string
	Template string
}

// AddJSONFlags adds --json, --jq, and --template flags to a command.
func AddJSONFlags(cmd *cobra.Command, flags *JSONFlags) {
	cmd.Flags().BoolVar(&flags.JSON, "json", false, "Output JSON")
	cmd.Flags().StringVar(&flags.JQ, "jq", "", "Filter JSON output using a jq expression")
	cmd.Flags().StringVar(&flags.Template, "template", "", "Format JSON output using a Go template")
}

// OutputJSON writes data as JSON, optionally filtered by jq or formatted by a template.
func (f *JSONFlags) OutputJSON(w io.Writer, data interface{}) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if f.JQ != "" {
		return applyJQ(w, jsonBytes, f.JQ)
	}

	if f.Template != "" {
		return applyTemplate(w, data, f.Template)
	}

	// Pretty print
	var out bytes.Buffer
	if err := json.Indent(&out, jsonBytes, "", "  "); err != nil {
		return err
	}
	_, err = out.WriteTo(w)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w)
	return err
}

// WantsJSON returns true if JSON output is requested (via --json, --jq, or --template).
func (f *JSONFlags) WantsJSON() bool {
	return f.JSON || f.JQ != "" || f.Template != ""
}

func applyJQ(w io.Writer, jsonBytes []byte, expr string) error {
	query, err := gojq.Parse(expr)
	if err != nil {
		return fmt.Errorf("invalid jq expression: %w", err)
	}

	var input interface{}
	if err := json.Unmarshal(jsonBytes, &input); err != nil {
		return fmt.Errorf("failed to parse JSON for jq: %w", err)
	}

	iter := query.Run(input)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, isErr := v.(error); isErr {
			return fmt.Errorf("jq error: %w", err)
		}

		out, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(w, string(out))
	}
	return nil
}

func applyTemplate(w io.Writer, data interface{}, tmplStr string) error {
	tmpl, err := template.New("output").Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("invalid template: %w", err)
	}
	return tmpl.Execute(w, data)
}
