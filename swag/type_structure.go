package swag

import (
	"fmt"
	"reflect"
	"strings"
)

type param interface {
	AcceptedKind() string
	IsQueryParam() bool
}

var paramInterface = reflect.TypeOf((*param)(nil)).Elem()

type TypeStructure struct {
	Name     string // if the type is a child of a struct or map, this is the field name or map key
	Kind     string // e.x. string, int, bool, struct, map, slice
	Nullable bool
	Children []TypeStructure
}

func (t TypeStructure) Format() string {
	switch t.Kind {
	case "int", "int8", "int16", "int32", "int64":
		return "integer"
	case "uint", "uint8", "uint16", "uint32", "uint64":
		return "unsigned integer"
	case "float", "float32", "float64":
		return "float"
	case "string":
		return "string"
	case "bool":
		return "boolean"
	case "struct":
		s := "{\n"
		for _, child := range t.Children {
			if child.Nullable {
				s += fmt.Sprintf("  \"%s\"?: %s\n", child.Name, padLines(child.Format(), 1))
			} else {
				s += fmt.Sprintf("  \"%s\": %s\n", child.Name, padLines(child.Format(), 1))
			}
		}
		s += "}"
		return s
	case "slice", "array":
		s := "Array<\n"
		for _, child := range t.Children {
			s += fmt.Sprintf("  %s\n", padLines(child.Format(), 1))
		}
		s += ">"
		return s
	case "map":
		s := "Map<string,\n"
		for _, child := range t.Children {
			s += fmt.Sprintf("  %s\n", padLines(child.Format(), 1))
		}
		s += ">"
		return s
	}

	return "unknown"
}

// NewTypeStructure generates a TypeStructure slice from a given interface{}.
// It recursively inspects the type and its fields/elements to build the structure.
func NewTypeStructure(input any) TypeStructure {
	if input == nil {
		return TypeStructure{} // Or a single TypeStructure representing nil
	}
	return generateTypeStructure(reflect.TypeOf(input), "", false)
}

func NewParamsTypeStructure(input any) TypeStructure {
	if input == nil {
		return TypeStructure{} // Or a single TypeStructure representing nil
	}
	return generateTypeStructure(reflect.TypeOf(input), "", true)
}

// generateTypeStructure is a helper function that recursively builds the TypeStructure.
func generateTypeStructure(t reflect.Type, name string, isParamsObject bool) TypeStructure {
	ts := TypeStructure{
		Name:     name,
		Kind:     t.Kind().String(),
		Nullable: isNullable(t),
	}

	switch t.Kind() {
	case reflect.Ptr:

		// special handling for the Params objects
		if isParamsObject {
			if t.Implements(paramInterface) {
				v := reflect.New(t.Elem())
				p := v.Interface().(param)
				ts.Kind = p.AcceptedKind()
				if p.IsQueryParam() {
					ts.Name = strings.ToLower(ts.Name)
					return ts
				}
			}

			return TypeStructure{}
		}

		// For pointers, we need to get the element type and recurse.
		// The nullable field is already set for the pointer itself.
		ts = generateTypeStructure(t.Elem(), name, isParamsObject)
		ts.Nullable = true
		return ts

	case reflect.Struct:
		// Iterate over struct fields
		for i := range t.NumField() {
			field := t.Field(i)
			// Skip unexported fields as reflection generally doesn't expose them for external inspection
			if field.PkgPath != "" { // PkgPath is empty for exported fields
				continue
			}
			name := field.Name
			jsonName := field.Tag.Get("json")
			if jsonName != "" {
				name = jsonName
			}
			if name != "-" {
				child := generateTypeStructure(field.Type, name, isParamsObject)
				if child.Kind != "" {
					ts.Children = append(ts.Children, child)
				}
			}
		}

	case reflect.Map:
		// For maps, we need to describe both the key and value types.
		// It's common to represent them as two children.
		key := generateTypeStructure(t.Key(), "key", isParamsObject)
		value := generateTypeStructure(t.Elem(), "value", isParamsObject)
		if value.Kind != "" && (key.Kind == "string" || isNumber(key.Kind)) {
			ts.Name = key.Kind
			value.Name = "element"
			ts.Children = append(ts.Children, value)
		}

	case reflect.Slice, reflect.Array:
		// For slices and arrays, describe the element type.
		child := generateTypeStructure(t.Elem(), "element", isParamsObject)
		if child.Kind != "" {
			ts.Children = append(ts.Children, child)
		}

	case reflect.Interface:
		// For interfaces, it's difficult to know the concrete type at compile time.
		// We can only state it's an interface.
		// If you need to inspect the concrete type an interface holds at runtime,
		// you'd typically need to pass the concrete value and then reflect on that.
	}

	return ts
}

func isNumber(k string) bool {
	return k == "int" || k == "int8" || k == "int32" || k == "int64" || k == "uint" || k == "uint8" || k == "uint16" || k == "uint32" || k == "uint64" || k == "float" || k == "float32" || k == "float64"
}

// isNullable checks if a reflect.Type is nullable.
// Pointers, interfaces, maps, slices, channels, and functions can be nil.
func isNullable(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func:
		return true
	default:
		return false
	}
}

func padLines(str string, startFromIdx int) string {
	lines := strings.Split(str, "\n")

	for i := startFromIdx; i < len(lines); i++ {
		lines[i] = "  " + lines[i]
	}

	return strings.Join(lines, "\n")
}
