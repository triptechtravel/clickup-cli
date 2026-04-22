package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"text/template"
)

// fixSelfRefs reads the generated types file, finds self-referencing type
// aliases (type X = []X), resolves them by following $ref chains in the
// spec, and writes a fixes file with proper struct definitions.
func fixSelfRefs(specPath, genFile, fixesFile, pkg string) error {
	// Step 1: Find self-referencing types in generated code.
	genData, err := os.ReadFile(genFile)
	if err != nil {
		return fmt.Errorf("read generated file: %w", err)
	}

	// Go's regexp doesn't support backreferences. Instead, find all
	// type aliases of form `type X = []Y` and check if X == Y.
	aliasRe := regexp.MustCompile(`(?m)^type (\w+) = \[\](\w+)$`)
	allAliases := aliasRe.FindAllStringSubmatch(string(genData), -1)

	var selfRefTypes []string
	for _, m := range allAliases {
		if m[1] == m[2] {
			selfRefTypes = append(selfRefTypes, m[1])
		}
	}
	if len(selfRefTypes) == 0 {
		fmt.Printf("No self-referencing types found in %s\n", genFile)
		return nil
	}
	sort.Strings(selfRefTypes)

	// Step 2: Load the spec as raw JSON for $ref resolution.
	specData, err := os.ReadFile(specPath)
	if err != nil {
		return fmt.Errorf("read spec: %w", err)
	}

	var rawSpec map[string]any
	if err := json.Unmarshal(specData, &rawSpec); err != nil {
		return fmt.Errorf("parse spec: %w", err)
	}

	// Step 3: For each self-referencing type, find it in the spec and
	// resolve what it should actually be.
	type resolvedType struct {
		Name   string
		Fields []fieldInfo
	}

	var resolved []resolvedType
	var unresolved []string

	for _, typeName := range selfRefTypes {
		// The type name encodes the spec path. Decode it:
		// GetV2SpaceSpaceIDTag200Response → paths["/v2/space/{space_id}/tag"].get.responses.200...
		// We need to find which response field references this type in the gen code.
		fields := resolveTypeFromSpec(rawSpec, string(genData), typeName)
		if fields != nil {
			resolved = append(resolved, resolvedType{Name: typeName, Fields: fields})
		} else {
			unresolved = append(unresolved, typeName)
		}
	}

	// Step 4: Find all undefined types (referenced but not defined).
	undefinedTypes := findUndefinedTypes(string(genData), selfRefTypes)

	// Step 5: Write the fixes file.
	f, err := os.Create(fixesFile)
	if err != nil {
		return fmt.Errorf("create fixes file: %w", err)
	}
	defer f.Close()

	err = fixesTmpl.Execute(f, map[string]any{
		"Pkg":        pkg,
		"Resolved":   resolved,
		"Unresolved": unresolved,
		"Undefined":  undefinedTypes,
	})
	if err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	// Step 6: Remove self-referencing type aliases from generated code
	// (they're now defined in fixes.go).
	// Re-read the gen file since we may need the latest content.
	updatedGen := string(genData)
	for _, typeName := range selfRefTypes {
		re := regexp.MustCompile(`(?m)^type ` + regexp.QuoteMeta(typeName) + ` = \[\]` + regexp.QuoteMeta(typeName) + `\n`)
		updatedGen = re.ReplaceAllString(updatedGen, "")
	}
	// Also fix null enum constants.
	nullRe := regexp.MustCompile(`(?m)(\w+) = null$`)
	updatedGen = nullRe.ReplaceAllString(updatedGen, "${1} = 0")

	if err := os.WriteFile(genFile, []byte(updatedGen), 0644); err != nil {
		return fmt.Errorf("write updated gen: %w", err)
	}

	fmt.Printf("Fixed %d self-referencing types, %d unresolved, %d undefined stubs → %s\n",
		len(resolved), len(unresolved), len(undefinedTypes), fixesFile)
	return nil
}

type fieldInfo struct {
	Name     string
	JSONName string
	GoType   string
}

// resolveTypeFromSpec finds the actual schema for a self-referencing type by:
// 1. Finding which struct field uses this type in the generated code
// 2. Searching the spec for an array property with that json field name
//    whose items have a $ref
// 3. Following the $ref to get the real schema with properties
func resolveTypeFromSpec(spec map[string]any, genCode, typeName string) []fieldInfo {
	// Step 1: Find the JSON field name that references this type in gen code.
	reStr := `\w+\s+\[\]` + regexp.QuoteMeta(typeName) + `\s+` + "`.+?json:\"([^\"]+)\"`"
	re := regexp.MustCompile(reStr)
	m := re.FindStringSubmatch(genCode)
	if m == nil {
		return nil
	}
	jsonFieldName := m[1]

	// Step 2: Search all spec paths for an array property with this field
	// name whose items have a $ref. Iterate in sorted order so the result
	// is deterministic — without sorting, Go's random map iteration can
	// pick different $refs across runs when multiple paths match.
	paths, ok := spec["paths"].(map[string]any)
	if !ok {
		return nil
	}

	pathNames := make([]string, 0, len(paths))
	for p := range paths {
		pathNames = append(pathNames, p)
	}
	sort.Strings(pathNames)

	for _, pathName := range pathNames {
		methodMap, ok := paths[pathName].(map[string]any)
		if !ok {
			continue
		}
		methodNames := make([]string, 0, len(methodMap))
		for mn := range methodMap {
			methodNames = append(methodNames, mn)
		}
		sort.Strings(methodNames)
		for _, methodName := range methodNames {
			opRaw := methodMap[methodName]
			if methodName == "parameters" {
				continue
			}
			opMap, ok := opRaw.(map[string]any)
			if !ok {
				continue
			}

			// Collect schemas from both responses and request bodies.
			var schemas []map[string]any
			if s := digMap(opMap, "responses", "200", "content", "application/json", "schema"); s != nil {
				schemas = append(schemas, s)
			}
			if s := digMap(opMap, "requestBody", "content", "application/json", "schema"); s != nil {
				schemas = append(schemas, s)
			}

			for _, schema := range schemas {
				props, ok := schema["properties"].(map[string]any)
				if !ok {
					continue
				}
				fieldSchema, ok := props[jsonFieldName].(map[string]any)
				if !ok {
					continue
				}
				if fieldSchema["type"] != "array" {
					continue
				}
				items, ok := fieldSchema["items"].(map[string]any)
				if !ok {
					continue
				}
				ref, ok := items["$ref"].(string)
				if !ok {
					continue
				}

				// Step 3: Follow the $ref to the actual schema.
				resolved := resolveRef(spec, ref)
				if resolved != nil {
					fields := extractFields(resolved)
					if len(fields) > 0 {
						return fields
					}
				}
			}
		}
	}

	return nil
}

// digMap navigates nested maps by key path.
func digMap(m map[string]any, keys ...string) map[string]any {
	current := m
	for _, key := range keys {
		next, ok := current[key].(map[string]any)
		if !ok {
			return nil
		}
		current = next
	}
	return current
}

// resolveRef follows a JSON Pointer $ref in the spec.
func resolveRef(root map[string]any, ref string) map[string]any {
	if !strings.HasPrefix(ref, "#/") {
		return nil
	}
	parts := strings.Split(ref[2:], "/")
	var current any = root
	for _, part := range parts {
		// Unescape JSON Pointer encoding.
		part = strings.ReplaceAll(part, "~1", "/")
		part = strings.ReplaceAll(part, "~0", "~")
		part, _ = url.PathUnescape(part)

		m, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = m[part]
		if current == nil {
			return nil
		}
	}
	if m, ok := current.(map[string]any); ok {
		return m
	}
	return nil
}

func extractFields(schema map[string]any) []fieldInfo {
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		return nil
	}
	var fields []fieldInfo
	for name, propRaw := range props {
		prop, ok := propRaw.(map[string]any)
		if !ok {
			continue
		}
		goType := schemaToGoType(prop)
		fields = append(fields, fieldInfo{
			Name:     toExportedName(name),
			JSONName: name,
			GoType:   goType,
		})
	}
	sort.Slice(fields, func(i, j int) bool { return fields[i].Name < fields[j].Name })
	return fields
}

func schemaToGoType(prop map[string]any) string {
	t, _ := prop["type"].(string)
	switch t {
	case "string":
		return "string"
	case "integer":
		return "int"
	case "number":
		return "float64"
	case "boolean":
		return "bool"
	case "array":
		return "[]any"
	case "object":
		return "any"
	default:
		return "any"
	}
}

func toExportedName(s string) string {
	parts := strings.Split(s, "_")
	for i := range parts {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	// Common abbreviations.
	result := strings.Join(parts, "")
	result = strings.ReplaceAll(result, "Id", "ID")
	result = strings.ReplaceAll(result, "Url", "URL")
	return result
}

func findUndefinedTypes(genCode string, excludeTypes []string) []string {
	exclude := make(map[string]bool)
	for _, t := range excludeTypes {
		exclude[t] = true
	}

	// Find all type definitions.
	defRe := regexp.MustCompile(`(?m)^type (\w+) `)
	defined := make(map[string]bool)
	for _, m := range defRe.FindAllStringSubmatch(genCode, -1) {
		defined[m[1]] = true
	}

	// Find all type references.
	refRe := regexp.MustCompile(`\b([A-Z]\w+(?:200|201|204)?Response(?:JSON\d+)?)\b`)
	referenced := make(map[string]bool)
	for _, m := range refRe.FindAllStringSubmatch(genCode, -1) {
		referenced[m[1]] = true
	}

	var undefined []string
	for ref := range referenced {
		if !defined[ref] && !exclude[ref] {
			undefined = append(undefined, ref)
		}
	}
	sort.Strings(undefined)
	return undefined
}

var fixesTmpl = template.Must(template.New("fixes").Parse(`// Code generated by gen-api (fixrefs); DO NOT EDIT.
//
// Provides proper type definitions for self-referencing aliases that the
// oapi-codegen-exp cannot resolve, plus stubs for missing types.

package {{ .Pkg }}
{{ range .Resolved }}
// {{ .Name }} — resolved from spec $ref chain.
type {{ .Name }} struct {
{{- range .Fields }}
	{{ .Name }} {{ .GoType }} ` + "`" + `json:"{{ .JSONName }}"` + "`" + `
{{- end }}
}
{{ end }}
{{- range .Unresolved }}
// {{ . }} — codegen could not resolve; using any.
type {{ . }} = any
{{ end }}
{{- range .Undefined }}
// {{ . }} — referenced but not defined by codegen.
type {{ . }} = any
{{ end }}
`))
