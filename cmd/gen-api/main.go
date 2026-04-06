// Command gen-api reads an OpenAPI spec and generates typed Go wrapper
// functions for each operation. The wrappers use the auto-generated types
// from clickupv2/clickupv3 packages and route through api.Client.
//
// Usage:
//
//	go run ./cmd/gen-api -spec api/specs/clickup-v2.json -pkg apiv2 -types-pkg clickupv2 -out internal/apiv2/operations.gen.go
//	go run ./cmd/gen-api -spec api/specs/clickup-v3.yaml -pkg apiv3 -types-pkg clickupv3 -out internal/apiv3/operations.gen.go
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"unicode"
)

type spec struct {
	Paths map[string]map[string]json.RawMessage `json:"paths"`
}

type operation struct {
	OperationID string    `json:"operationId"`
	Summary     string    `json:"summary"`
	Parameters  []param   `json:"parameters"`
	RequestBody *struct {
		Content map[string]struct {
			Schema struct {
				Ref string `json:"$ref"`
			} `json:"schema"`
		} `json:"content"`
	} `json:"requestBody"`
	Responses map[string]struct {
		Content map[string]struct {
			Schema struct {
				Ref string `json:"$ref"`
			} `json:"schema"`
		} `json:"content"`
	} `json:"responses"`
}

type param struct {
	Name     string `json:"name"`
	In       string `json:"in"`
	Required bool   `json:"required"`
	Schema   struct {
		Type string `json:"type"`
		Ref  string `json:"$ref"`
	} `json:"schema"`
}

type opInfo struct {
	FuncName    string
	Method      string
	PathPattern string // e.g., "task/%s/comment"
	PathArgs    []pathArg
	QueryParams []queryParam
	HasReqBody  bool
	HasRespBody bool
	ReqType     string // e.g., "clickupv2.UpdateTaskJSONRequest"
	RespType    string // e.g., "clickupv2.GetTaskJSONResponse"
	Summary     string
}

type pathArg struct {
	Name     string // Go param name
	SpecName string // original spec name
}

type queryParam struct {
	Name     string
	SpecName string
	Type     string
	Required bool
}

var (
	specPath = flag.String("spec", "", "path to OpenAPI spec (JSON)")
	pkg      = flag.String("pkg", "apiv2", "output Go package name")
	typesPkg = flag.String("types-pkg", "clickupv2", "types package name")
	typesImp = flag.String("types-import", "", "types package import path (auto-detected if empty)")
	outPath  = flag.String("out", "", "output file path")
	module   = flag.String("module", "github.com/triptechtravel/clickup-cli", "Go module path")
	fixMode  = flag.Bool("fix", false, "fix self-referencing types in generated code instead of generating wrappers")
	fixGen   = flag.String("fix-gen", "", "path to generated types file (for -fix mode)")
	fixOut   = flag.String("fix-out", "", "path to write fixes file (for -fix mode)")
	fixPkg   = flag.String("fix-pkg", "", "package name for fixes file (for -fix mode)")
)

var pathParamRe = regexp.MustCompile(`\{([^}]+)\}`)

func main() {
	flag.Parse()

	if *fixMode {
		if *specPath == "" || *fixGen == "" || *fixOut == "" || *fixPkg == "" {
			log.Fatal("-fix mode requires -spec, -fix-gen, -fix-out, and -fix-pkg")
		}
		if err := fixSelfRefs(*specPath, *fixGen, *fixOut, *fixPkg); err != nil {
			log.Fatalf("fix: %v", err)
		}
		return
	}

	if *specPath == "" || *outPath == "" {
		flag.Usage()
		os.Exit(1)
	}

	data, err := os.ReadFile(*specPath)
	if err != nil {
		log.Fatalf("read spec: %v", err)
	}

	var s spec
	if err := json.Unmarshal(data, &s); err != nil {
		log.Fatalf("parse spec: %v", err)
	}

	// Also parse as raw JSON for $ref resolution.
	var rawSpec map[string]any
	json.Unmarshal(data, &rawSpec)

	if *typesImp == "" {
		*typesImp = *module + "/api/" + *typesPkg
	}

	// Read generated types file to know which types actually exist.
	typesFile := "api/" + *typesPkg + "/client.gen.go"
	existingTypes := readExistingTypes(typesFile)

	ops := extractOperations(s, *typesPkg, existingTypes, rawSpec)
	sort.Slice(ops, func(i, j int) bool { return ops[i].FuncName < ops[j].FuncName })

	if err := writeFile(*outPath, ops, existingTypes); err != nil {
		log.Fatalf("write: %v", err)
	}

	fmt.Printf("Generated %d operations → %s\n", len(ops), *outPath)
}

func readExistingTypes(path string) map[string]bool {
	types := make(map[string]bool)
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("warning: cannot read types file %s: %v", path, err)
		return types
	}
	// Match "type FooBar struct" or "type FooBar = ..."
	re := regexp.MustCompile(`(?m)^type (\w+) `)
	for _, m := range re.FindAllStringSubmatch(string(data), -1) {
		types[m[1]] = true
	}
	return types
}

func extractOperations(s spec, typesPkg string, existingTypes map[string]bool, rawSpec map[string]any) []opInfo {
	var ops []opInfo

	for path, methods := range s.Paths {
		for method, raw := range methods {
			if method == "parameters" {
				continue
			}
			var op operation
			if err := json.Unmarshal(raw, &op); err != nil {
				continue // skip non-operation entries
			}
			if op.OperationID == "" {
				continue
			}

			info := opInfo{
				FuncName:   cleanFuncName(op.OperationID),
				Method:     strings.ToUpper(method),
				Summary:    op.Summary,
				HasReqBody: op.RequestBody != nil,
			}

			// Check if 200 or 201 response has JSON content.
			for _, code := range []string{"200", "201"} {
				if resp, ok := op.Responses[code]; ok {
					if _, ok := resp.Content["application/json"]; ok {
						info.HasRespBody = true
						break
					}
				}
			}

			// Build path pattern: /v2/task/{task_id} → "task/%s"
			// Also handle V3 paths: /api/v3/workspaces/{id}/docs → "workspaces/%s/docs"
			trimmed := path
			for _, prefix := range []string{"/v2/", "/api/v3/", "/api/v2/"} {
				if strings.HasPrefix(trimmed, prefix) {
					trimmed = strings.TrimPrefix(trimmed, prefix)
					break
				}
			}
			var pathArgs []pathArg
			pattern := pathParamRe.ReplaceAllStringFunc(trimmed, func(match string) string {
				name := match[1 : len(match)-1]
				pathArgs = append(pathArgs, pathArg{
					Name:     toCamelParam(name),
					SpecName: name,
				})
				return "%s"
			})
			info.PathPattern = pattern
			info.PathArgs = pathArgs

			// Extract query params, resolving $ref for types.
			for _, p := range op.Parameters {
				if p.In == "query" {
					paramType := p.Schema.Type
					if paramType == "" && p.Schema.Ref != "" && rawSpec != nil {
						// Follow $ref to component schema to get the type.
						// e.g., "#/components/schemas/FooQuery" → type alias = float32
						refName := refToTypeName(p.Schema.Ref)
						if comps, ok := rawSpec["components"].(map[string]any); ok {
							if schemas, ok := comps["schemas"].(map[string]any); ok {
								if schema, ok := schemas[refName].(map[string]any); ok {
									if t, ok := schema["type"].(string); ok {
										paramType = t
									}
								}
							}
						}
					}
					info.QueryParams = append(info.QueryParams, queryParam{
						Name:     toCamelParam(p.Name),
						SpecName: p.Name,
						Type:     goType(paramType),
						Required: p.Required,
					})
				}
			}

			// Map to auto-gen type names. Strategy:
			// 1. Try {OperationId}JSONRequest/Response (V2 inline schema pattern)
			// 2. Follow $ref to components/schemas and use that type name (V3 pattern)
			// 3. Fall back to skipping request or dropping response
			cleanedOpID := cleanFuncName(op.OperationID)
			skip := false

			if info.HasReqBody {
				info.ReqType = resolveTypeNameWithTypes(typesPkg, existingTypes, cleanedOpID+"JSONRequest", op.OperationID+"JSONRequest",
					refToTypeName(op.RequestBody.Content["application/json"].Schema.Ref))
				if info.ReqType == "" {
					skip = true
				}
			}
			if info.HasRespBody {
				var respRef string
				for _, code := range []string{"200", "201"} {
					if resp, ok := op.Responses[code]; ok {
						if ct, ok := resp.Content["application/json"]; ok {
							if ct.Schema.Ref != "" {
								respRef = ct.Schema.Ref
								break
							}
						}
					}
				}
				info.RespType = resolveTypeNameWithTypes(typesPkg, existingTypes, cleanedOpID+"JSONResponse", op.OperationID+"JSONResponse",
					refToTypeName(respRef))
				if info.RespType == "" {
					info.HasRespBody = false
				}
			}

			if skip {
				continue
			}

			ops = append(ops, info)
		}
	}

	return ops
}

// refToTypeName extracts a Go type name from a $ref like
// "#/components/schemas/PublicDocsDocCoreDto" → "PublicDocsDocCoreDto"
func refToTypeName(ref string) string {
	if ref == "" {
		return ""
	}
	parts := strings.Split(ref, "/")
	return parts[len(parts)-1]
}

// resolveTypeNameWithTypes tries candidate names against existingTypes and returns
// the first match prefixed with the types package.
func resolveTypeNameWithTypes(typesPkg string, existingTypes map[string]bool, candidates ...string) string {
	for _, name := range candidates {
		if name == "" {
			continue
		}
		if existingTypes[name] {
			return typesPkg + "." + name
		}
	}
	return ""
}

func cleanFuncName(opID string) string {
	// Remove apostrophes, spaces, etc.
	cleaned := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return -1
	}, opID)
	if cleaned == "" {
		return opID
	}
	// Ensure first letter is uppercase.
	return strings.ToUpper(cleaned[:1]) + cleaned[1:]
}

func toCamelParam(s string) string {
	parts := strings.Split(s, "_")
	for i := range parts {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	result := strings.Join(parts, "")
	// Lowercase first letter for param names.
	if len(result) > 0 {
		result = strings.ToLower(result[:1]) + result[1:]
	}
	return result
}

func goType(schemaType string) string {
	switch schemaType {
	case "integer":
		return "int"
	case "number":
		return "float64"
	case "boolean":
		return "bool"
	default:
		return "string"
	}
}

func writeFile(path string, ops []opInfo, existingTypes map[string]bool) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return tmpl.Execute(f, map[string]any{
		"Pkg":         *pkg,
		"TypesPkg":    *typesPkg,
		"TypesImport": *typesImp,
		"Ops":         ops,
		"HasNullable": existingTypes["Nullable"],
	})
}

var tmpl = template.Must(template.New("").Funcs(template.FuncMap{
	"upper":         strings.ToUpper,
	"hasQueryParams": func(qp []queryParam) bool { return len(qp) > 0 },
	"exportName": func(s string) string {
		parts := strings.Split(s, "_")
		for i := range parts {
			if len(parts[i]) > 0 {
				parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
			}
		}
		r := strings.Join(parts, "")
		r = strings.ReplaceAll(r, "[]", "")
		return r
	},
}).Parse(`// Code generated by gen-api from OpenAPI spec; DO NOT EDIT.

package {{ .Pkg }}

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/triptechtravel/clickup-cli/internal/api"
	{{ .TypesPkg }} "{{ .TypesImport }}"
)

// Ensure imports are used.
var _ = fmt.Sprintf
var _ = url.Values{}
var _ = strconv.Itoa
{{- if .HasNullable }}
var _ {{ .TypesPkg }}.Nullable[string]
{{- end }}
{{ range .Ops }}{{ if hasQueryParams .QueryParams }}
// {{ .FuncName }}Params holds optional query parameters for {{ .FuncName }}.
type {{ .FuncName }}Params struct {
{{- range .QueryParams }}
	{{ exportName .SpecName }} {{ .Type }} ` + "`" + `url:"{{ .SpecName }},omitempty"` + "`" + `
{{- end }}
}

func (p {{ .FuncName }}Params) encode() string {
	q := url.Values{}
{{- range .QueryParams }}
{{- if eq .Type "string" }}
	if p.{{ exportName .SpecName }} != "" { q.Set("{{ .SpecName }}", p.{{ exportName .SpecName }}) }
{{- else if eq .Type "int" }}
	if p.{{ exportName .SpecName }} != 0 { q.Set("{{ .SpecName }}", strconv.Itoa(p.{{ exportName .SpecName }})) }
{{- else if eq .Type "float64" }}
	if p.{{ exportName .SpecName }} != 0 { q.Set("{{ .SpecName }}", fmt.Sprintf("%g", p.{{ exportName .SpecName }})) }
{{- else if eq .Type "bool" }}
	if p.{{ exportName .SpecName }} { q.Set("{{ .SpecName }}", "true") }
{{- else }}
	if p.{{ exportName .SpecName }} != "" { q.Set("{{ .SpecName }}", fmt.Sprint(p.{{ exportName .SpecName }})) }
{{- end }}
{{- end }}
	if len(q) == 0 { return "" }
	return "?" + q.Encode()
}
{{ end }}
// {{ .FuncName }} — {{ .Summary }}
// {{ .Method }} /{{ .PathPattern }}
func {{ .FuncName }}(ctx context.Context, client *api.Client{{ range .PathArgs }}, {{ .Name }} string{{ end }}{{ if .HasReqBody }}, req *{{ .ReqType }}{{ end }}{{ if hasQueryParams .QueryParams }}, opts ...{{ .FuncName }}Params{{ end }}) {{ if .HasRespBody }}(*{{ .RespType }}, error){{ else }}error{{ end }} {
	{{- if .HasRespBody }}
	var resp {{ .RespType }}
	{{- end }}
	path := {{ if .PathArgs }}fmt.Sprintf("{{ .PathPattern }}"{{ range .PathArgs }}, {{ .Name }}{{ end }}){{ else }}"{{ .PathPattern }}"{{ end }}{{ if hasQueryParams .QueryParams }}
	if len(opts) > 0 { path += opts[0].encode() }{{ end }}
	{{- if .HasRespBody }}
	if err := do(ctx, client, "{{ .Method }}", path, {{ if .HasReqBody }}req{{ else }}nil{{ end }}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
	{{- else }}
	return do(ctx, client, "{{ .Method }}", path, {{ if .HasReqBody }}req{{ else }}nil{{ end }}, nil)
	{{- end }}
}
{{ end }}
`))
