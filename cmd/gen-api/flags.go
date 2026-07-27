package main

// Flag generation.
//
// Commands used to hand-write one flag per request field. That is how
// `archived` — present in the spec and in the generated request type — never
// reached the CLI, and it is why archiving 60 tasks needed raw curl.
//
// This emits, for every operation with a request body, a Flags struct that
// registers a cobra flag per scalar field and applies the changed ones to the
// generated request. A new field in the spec becomes a working flag after
// `make api-gen`, with no Go written.
//
// Scope: scalar fields only (string/boolean/integer/number). Objects, arrays
// and nullable unions — assignees, custom fields, time_estimate — need bespoke
// parsing and value semantics, so they stay hand-written. Register(skip...)
// lets a command opt out of any generated flag it handles better itself, which
// is what makes adoption incremental rather than all-or-nothing.

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/template"
)

// flagField is one generated flag.
type flagField struct {
	GoName    string // Archived
	FlagName  string // archived
	GoType    string // bool
	FlagFunc  string // BoolVar
	Doc       string
	IsBoolean bool
	IsPointer bool // optional in the spec => pointer in the generated type
}

// flagOp is one operation's generated flag set.
type flagOp struct {
	FuncName string // UpdateTask
	ReqType  string // clickupv2.UpdateTaskJSONRequest
	Fields   []flagField
}

// schemaProps is the minimal shape needed to read inline request schemas.
type schemaProps struct {
	Properties map[string]struct {
		Type        any    `json:"type"`
		Description string `json:"description"`
	} `json:"properties"`
	// Required fields are emitted as plain values by oapi-codegen; optional
	// ones as pointers. Apply has to assign accordingly.
	Required []string `json:"required"`
}

// scalarKind maps an OpenAPI scalar type to Go. Non-scalars return "".
func scalarKind(t any) (goType, flagFunc string) {
	s, ok := t.(string)
	if !ok {
		// Union types such as ["integer","null"] need nullable semantics the
		// generated wrappers model separately; skip rather than guess.
		return "", ""
	}
	switch s {
	case "string":
		return "string", "StringVar"
	case "boolean":
		return "bool", "BoolVar"
	case "integer":
		return "int", "IntVar"
	case "number":
		return "float32", "Float32Var"
	}
	return "", ""
}

// collectFlagOps reads inline request-body schemas and builds the flag sets.
func collectFlagOps(rawSpec []byte, ops []opInfo, typesPkg string) ([]flagOp, error) {
	// A path object holds operations keyed by method, but also non-operation
	// keys such as a shared "parameters" array — so decode lazily per key.
	type specOp struct {
		OperationID string `json:"operationId"`
		RequestBody *struct {
			Content map[string]struct {
				Schema schemaProps `json:"schema"`
			} `json:"content"`
		} `json:"requestBody"`
	}
	var doc struct {
		Paths map[string]map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal(rawSpec, &doc); err != nil {
		return nil, fmt.Errorf("parse spec for flags: %w", err)
	}
	httpMethods := map[string]bool{"get": true, "put": true, "post": true, "patch": true, "delete": true}

	// operationId -> generated wrapper name, so emitted flags line up with the
	// operations they feed.
	byFunc := map[string]opInfo{}
	for _, op := range ops {
		byFunc[op.FuncName] = op
	}

	var out []flagOp
	for _, methods := range doc.Paths {
		for method, raw := range methods {
			if !httpMethods[strings.ToLower(method)] {
				continue
			}
			var op specOp
			if err := json.Unmarshal(raw, &op); err != nil {
				continue
			}
			if op.RequestBody == nil || op.OperationID == "" {
				continue
			}
			funcName := cleanFuncName(op.OperationID)
			info, ok := byFunc[funcName]
			if !ok || !info.HasReqBody {
				continue
			}
			content, ok := op.RequestBody.Content["application/json"]
			if !ok {
				continue
			}

			required := map[string]bool{}
			for _, r := range content.Schema.Required {
				required[r] = true
			}

			var fields []flagField
			for name, prop := range content.Schema.Properties {
				goType, flagFunc := scalarKind(prop.Type)
				if goType == "" {
					continue
				}
				fields = append(fields, flagField{
					GoName:    goFieldName(name),
					FlagName:  strings.ReplaceAll(name, "_", "-"),
					GoType:    goType,
					FlagFunc:  flagFunc,
					Doc:       firstLine(prop.Description),
					IsBoolean: goType == "bool",
					IsPointer: !required[name],
				})
			}
			if len(fields) == 0 {
				continue
			}
			sort.Slice(fields, func(i, j int) bool { return fields[i].GoName < fields[j].GoName })

			out = append(out, flagOp{
				FuncName: funcName,
				ReqType:  typesPkg + "." + strings.TrimPrefix(info.ReqType, typesPkg+"."),
				Fields:   fields,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].FuncName < out[j].FuncName })
	return out, nil
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, ".\n"); i > 0 {
		s = s[:i]
	}
	s = strings.ReplaceAll(s, "`", "")
	if len(s) > 90 {
		s = s[:90]
	}
	return s
}

func writeFlags(path, pkg, typesImport string, ops []flagOp) (err error) {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	// Close is reported: a failed flush here would silently truncate generated
	// code, and the next build would fail somewhere unrelated.
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close %s: %w", path, cerr)
		}
	}()
	return flagsTmpl.Execute(f, map[string]any{
		"Pkg":         pkg,
		"TypesImport": typesImport,
		"Ops":         ops,
	})
}

var flagsTmpl = template.Must(template.New("flags").Funcs(template.FuncMap{
	"q": strconv.Quote,
}).Parse(`// Code generated by gen-api. DO NOT EDIT.
//
// One Flags struct per operation with a request body. Register binds a cobra
// flag per scalar request field; Apply copies the ones the user actually set.
//
// A field added to the spec becomes a flag here after ` + "`make api-gen`" + `,
// with no hand-written Go. Fields a command handles itself (fuzzy status
// matching, assignee add/remove, date parsing) are excluded via Register's
// skip list.
package {{.Pkg}}

import (
	"github.com/spf13/cobra"
	"{{.TypesImport}}"
)
{{range .Ops}}
// {{.FuncName}}Flags binds request fields for {{.FuncName}}.
type {{.FuncName}}Flags struct {
	flags *cobra.Command
{{- range .Fields}}
	{{.GoName}} {{.GoType}}
{{- end}}
}

// Register adds a flag per scalar request field. Names in skip are omitted, for
// fields the command binds itself.
func (f *{{.FuncName}}Flags) Register(cmd *cobra.Command, skip ...string) {
	f.flags = cmd
	skipped := map[string]bool{}
	for _, s := range skip {
		skipped[s] = true
	}
{{- range .Fields}}
	if !skipped["{{.FlagName}}"] {
		cmd.Flags().{{.FlagFunc}}(&f.{{.GoName}}, {{q .FlagName}}, {{if .IsBoolean}}false{{else if eq .GoType "string"}}""{{else}}0{{end}}, {{if .Doc}}{{q .Doc}}{{else}}{{q (printf "Set %s" .FlagName)}}{{end}})
	}
{{- end}}
}

// Apply copies every flag the user actually changed onto req. Unset flags are
// left nil so they are omitted from the request body rather than sent as zero.
func (f *{{.FuncName}}Flags) Apply(req *{{.ReqType}}) {
	if f.flags == nil {
		return
	}
{{- range .Fields}}
	if f.flags.Flags().Changed("{{.FlagName}}") {
{{- if .IsPointer}}
		v := f.{{.GoName}}
		req.{{.GoName}} = &v
{{- else}}
		req.{{.GoName}} = f.{{.GoName}}
{{- end}}
	}
{{- end}}
}
{{end}}`))

// goFieldName converts a snake_case spec property to the exported Go field
// name oapi-codegen produces (markdown_content -> MarkdownContent).
func goFieldName(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if p == "" {
			continue
		}
		switch strings.ToLower(p) {
		case "id":
			parts[i] = "ID"
		case "url":
			parts[i] = "URL"
		default:
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}
