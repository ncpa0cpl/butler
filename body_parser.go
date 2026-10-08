package butler

import (
	"bytes"
	"encoding"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	echo "github.com/labstack/echo/v5"
)

// bodyBindOptions controls how required fields are validated while the request
// body is being bound.
type bodyBindOptions struct {
	// When true every bindable field of the body struct must be present in
	// the request body, otherwise a Bad Request error is returned.
	requireAllFields bool
}

func bindBody(c *echo.Context, target any, opts bodyBindOptions) (err error) {
	req := c.Request()
	if req.ContentLength == 0 {
		// with no body every required field is necessarily missing
		return validateRequiredZeroFields(target, opts.requireAllFields)
	}

	// mediatype is found like `mime.ParseMediaType()` does it
	base, _, _ := strings.Cut(req.Header.Get(echo.HeaderContentType), ";")
	mediatype := strings.TrimSpace(base)

	switch mediatype {
	case echo.MIMEApplicationJSON:
		// read the whole body so it can be re-inspected for missing fields
		// after binding, leaving a replacement reader for the serializer
		body, rerr := io.ReadAll(req.Body)
		if rerr != nil {
			return echo.ErrBadRequest.Wrap(rerr)
		}
		req.Body = io.NopCloser(bytes.NewReader(body))

		if err = c.Echo().JSONSerializer.Deserialize(c, target); err != nil {
			var hErr *echo.HTTPError
			if errors.As(err, &hErr) {
				return err
			}
			return echo.ErrBadRequest.Wrap(err)
		}

		if opts.requireAllFields || structHasRequiredTags(derefType(reflect.TypeOf(target))) {
			if err = checkJSONRequiredFields(target, body, opts.requireAllFields); err != nil {
				return echo.ErrBadRequest.Wrap(err)
			}
		}
	case echo.MIMEApplicationXML, echo.MIMETextXML:
		if err = xml.NewDecoder(req.Body).Decode(target); err != nil {
			return echo.ErrBadRequest.Wrap(err)
		}
		// the raw XML keys are gone after decoding, so fall back to
		// zero-value checks
		if err = validateRequiredZeroFields(target, opts.requireAllFields); err != nil {
			return echo.ErrBadRequest.Wrap(err)
		}
	case echo.MIMEApplicationForm:
		params, err := c.FormValues()
		if err != nil {
			return echo.ErrBadRequest.Wrap(err)
		}
		if err = bindData(target, params, "form", nil, opts); err != nil {
			return echo.ErrBadRequest.Wrap(err)
		}
	case echo.MIMEMultipartForm:
		params, err := c.MultipartForm()
		if err != nil {
			return echo.ErrBadRequest.Wrap(err)
		}
		if err = bindData(target, params.Value, "form", params.File, opts); err != nil {
			return echo.ErrBadRequest.Wrap(err)
		}
	default:
		return &echo.HTTPError{Code: http.StatusUnsupportedMediaType}
	}
	return nil
}

// A field counts as present when its key exists in the JSON object and its
// value is not `null`, so `{"age": 0}` satisfies a required `Age` field.
func checkJSONRequiredFields(target any, body []byte, requireAll bool) error {
	typ := reflect.TypeOf(target)
	if typ == nil {
		return nil
	}
	typ = derefType(typ)
	if typ.Kind() != reflect.Struct {
		return nil
	}

	var raw map[string]json.RawMessage
	if len(body) > 0 {
		// a non-object body (or a `null` one) just means nothing is present
		_ = json.Unmarshal(body, &raw)
	}

	var missing []string
	checkJSONStructFields(typ, raw, requireAll, "", &missing)
	if len(missing) > 0 {
		return missingRequiredFieldsError(missing)
	}
	return nil
}

func checkJSONStructFields(typ reflect.Type, raw map[string]json.RawMessage, requireAll bool, path string, missing *[]string) {
	for i := 0; i < typ.NumField(); i++ {
		typeField := typ.Field(i)
		// embedded unexported structs can still bind promoted fields
		if typeField.PkgPath != "" && !typeField.Anonymous {
			continue
		}

		jsonTag, hasJSONTag := typeField.Tag.Lookup("json")
		name := typeField.Name
		hasExplicitName := false
		if hasJSONTag {
			name = strings.TrimSpace(strings.Split(jsonTag, ",")[0])
			if name == "-" {
				continue
			}
			hasExplicitName = name != ""
		}

		// embedded structs without a JSON name are promoted into the
		// parent object, so check them against the same map
		if typeField.Anonymous && !hasExplicitName {
			if fieldType := derefType(typeField.Type); fieldType.Kind() == reflect.Struct && !isOpaqueStruct(fieldType) {
				checkJSONStructFields(fieldType, raw, requireAll, path, missing)
				continue
			}
		}

		rawValue, found := lookupJSONKey(raw, name)
		if found && isJSONNull(rawValue) {
			found = false
		}

		if (requireAll || isRequiredTag(typeField.Tag)) && !found {
			*missing = append(*missing, path+name)
		}

		fieldType := derefType(typeField.Type)
		checkInto := func(elemType reflect.Type, elemPath string, value json.RawMessage) {
			var child map[string]json.RawMessage
			if json.Unmarshal(value, &child) == nil && child != nil {
				checkJSONStructFields(elemType, child, requireAll, elemPath, missing)
			}
		}
		switch {
		case fieldType.Kind() == reflect.Struct:
			if isOpaqueStruct(fieldType) || !(requireAll || structHasRequiredTags(fieldType)) || !found {
				continue
			}
			checkInto(fieldType, path+name+".", rawValue)
		case fieldType.Kind() == reflect.Slice || fieldType.Kind() == reflect.Array:
			elemType := derefType(fieldType.Elem())
			if elemType.Kind() != reflect.Struct || isOpaqueStruct(elemType) || !(requireAll || structHasRequiredTags(elemType)) || !found {
				continue
			}
			var elements []json.RawMessage
			if json.Unmarshal(rawValue, &elements) != nil {
				continue
			}
			for j, element := range elements {
				checkInto(elemType, fmt.Sprintf("%s%s[%d].", path, name, j), element)
			}
		}
	}
}

// Used where raw keys are unavailable (XML, empty body): a field left at its
// zero value is reported as missing, even if it was actually sent.
func validateRequiredZeroFields(target any, requireAll bool) error {
	typ := reflect.TypeOf(target)
	if typ == nil || typ.Kind() != reflect.Pointer || typ.Elem().Kind() != reflect.Struct {
		return nil
	}

	var missing []string
	walkRequiredZeroFields(typ.Elem(), reflect.ValueOf(target).Elem(), requireAll, "", &missing)
	if len(missing) > 0 {
		return missingRequiredFieldsError(missing)
	}
	return nil
}

func walkRequiredZeroFields(typ reflect.Type, val reflect.Value, requireAll bool, path string, missing *[]string) {
	for i := 0; i < typ.NumField(); i++ {
		typeField := typ.Field(i)
		structField := val.Field(i)
		if !structField.CanSet() {
			continue
		}

		name := bodyFieldName(typeField)
		required := requireAll || isRequiredTag(typeField.Tag)
		fieldType := derefType(structField.Type())
		if fieldType.Kind() == reflect.Struct && !isOpaqueStruct(fieldType) {
			if structField.IsZero() {
				if required {
					*missing = append(*missing, path+name)
				}
				continue
			}
			if structField.Kind() == reflect.Pointer {
				structField = structField.Elem()
			}
			walkRequiredZeroFields(fieldType, structField, requireAll, path+name+".", missing)
			continue
		}

		if required && structField.IsZero() {
			*missing = append(*missing, path+name)
		}
	}
}

// true when the struct or any struct nested inside it has a field marked
// required
func structHasRequiredTags(typ reflect.Type) bool {
	return structHasRequiredTagsRec(typ, map[reflect.Type]bool{})
}

func structHasRequiredTagsRec(typ reflect.Type, visiting map[reflect.Type]bool) bool {
	if typ.Kind() != reflect.Struct || visiting[typ] {
		return false
	}
	visiting[typ] = true
	defer delete(visiting, typ)

	for i := 0; i < typ.NumField(); i++ {
		typeField := typ.Field(i)
		if isRequiredTag(typeField.Tag) {
			return true
		}
		if fieldType := derefType(typeField.Type); fieldType.Kind() == reflect.Struct && !isOpaqueStruct(fieldType) {
			if structHasRequiredTagsRec(fieldType, visiting) {
				return true
			}
		}
	}
	return false
}

// a bare `required:""` tag also counts as true
func isRequiredTag(tag reflect.StructTag) bool {
	value, ok := tag.Lookup("required")
	if !ok {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "true", "1", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func missingRequiredFieldsError(missing []string) error {
	return echo.ErrBadRequest.Wrap(fmt.Errorf("missing required body field(s): %s", strings.Join(missing, ", ")))
}

func lookupJSONKey(raw map[string]json.RawMessage, name string) (json.RawMessage, bool) {
	if value, ok := raw[name]; ok {
		return value, true
	}
	// encoding/json matches field names case-insensitively, so do the same here
	for k, v := range raw {
		if strings.EqualFold(k, name) {
			return v, true
		}
	}
	return nil, false
}

func isJSONNull(value json.RawMessage) bool {
	return string(value) == "null"
}

func derefType(typ reflect.Type) reflect.Type {
	for typ != nil && typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	return typ
}

// best-effort body name, only used for error messages
func bodyFieldName(field reflect.StructField) string {
	for _, tag := range []string{"json", "xml", "form", "query", "param"} {
		if value, ok := field.Tag.Lookup(tag); ok {
			name := strings.TrimSpace(strings.Split(value, ",")[0])
			if name != "" && name != "-" {
				return name
			}
		}
	}
	return field.Name
}

var (
	opaqueStructTypes = map[reflect.Type]bool{
		// values whose subfields say nothing about presence
		reflect.TypeFor[time.Time]():            true,
		reflect.TypeFor[multipart.FileHeader](): true,
	}
	jsonUnmarshalerType = reflect.TypeFor[json.Unmarshaler]()
	textUnmarshalerType = reflect.TypeFor[encoding.TextUnmarshaler]()
	bindUnmarshalerType = reflect.TypeFor[echo.BindUnmarshaler]()
)

// custom-unmarshalable types are treated as single values, not recursed into
func isOpaqueStruct(typ reflect.Type) bool {
	if opaqueStructTypes[typ] {
		return true
	}
	pointerType := reflect.PointerTo(typ)
	return pointerType.Implements(jsonUnmarshalerType) ||
		pointerType.Implements(textUnmarshalerType) ||
		pointerType.Implements(bindUnmarshalerType)
}

// bindData will bind data ONLY fields in destination struct that have EXPLICIT tag
func bindData(destination any, data map[string][]string, tag string, dataFiles map[string][]*multipart.FileHeader, opts bodyBindOptions) error {
	if destination == nil || (len(data) == 0 && len(dataFiles) == 0) {
		return nil
	}
	hasFiles := len(dataFiles) > 0
	typ := reflect.TypeOf(destination).Elem()
	val := reflect.ValueOf(destination).Elem()

	// Support binding to limited Map destinations:
	// - map[string][]string,
	// - map[string]string <-- (binds first value from data slice)
	// - map[string]any
	// You are better off binding to struct but there are user who want this map feature. Source of data for these cases are:
	// params,query,header,form as these sources produce string values, most of the time slice of strings, actually.
	if typ.Kind() == reflect.Map && typ.Key().Kind() == reflect.String {
		k := typ.Elem().Kind()
		isElemInterface := k == reflect.Interface
		isElemString := k == reflect.String
		isElemSliceOfStrings := k == reflect.Slice && typ.Elem().Elem().Kind() == reflect.String
		if !(isElemSliceOfStrings || isElemString || isElemInterface) {
			return nil
		}
		if val.IsNil() {
			val.Set(reflect.MakeMap(typ))
		}
		for k, v := range data {
			if isElemString {
				val.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(v[0]))
			} else if isElemInterface {
				// To maintain backward compatibility, we always bind to the first string value
				// and not the slice of strings when dealing with map[string]any{}
				val.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(v[0]))
			} else {
				val.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(v))
			}
		}
		return nil
	}

	// !struct
	if typ.Kind() != reflect.Struct {
		if tag == "param" || tag == "query" || tag == "header" {
			// incompatible type, data is probably to be found in the body
			return nil
		}
		return errors.New("binding element must be a struct")
	}

	var missing []string
	for i := 0; i < typ.NumField(); i++ { // iterate over all destination fields
		typeField := typ.Field(i)
		structField := val.Field(i)
		if typeField.Anonymous {
			if structField.Kind() == reflect.Ptr {
				structField = structField.Elem()
			}
		}
		if !structField.CanSet() {
			continue
		}
		structFieldKind := structField.Kind()
		inputFieldName := typeField.Tag.Get(tag)
		if typeField.Anonymous && structFieldKind == reflect.Struct && inputFieldName != "" {
			// if anonymous struct with query/param/form tags, report an error
			return errors.New("query/param/form tags are not allowed with anonymous struct field")
		}

		if inputFieldName == "" {
			// If tag is nil, we inspect if the field is a not BindUnmarshaler struct and try to bind data into it (might contain fields with tags).
			// structs that implement BindUnmarshaler are bound only when they have explicit tag
			if _, ok := structField.Addr().Interface().(echo.BindUnmarshaler); !ok && structFieldKind == reflect.Struct {
				if err := bindData(structField.Addr().Interface(), data, tag, dataFiles, opts); err != nil {
					return err
				}
			}
			// does not have explicit tag and is not an ordinary struct - so move to next field
			continue
		}

		if hasFiles {
			if ok, err := isFieldMultipartFile(structField.Type()); err != nil {
				return err
			} else if ok {
				if ok := setMultipartFileHeaderTypes(structField, inputFieldName, dataFiles); ok {
					continue
				}
			}
		}

		inputValue, exists := data[inputFieldName]
		if !exists {
			// Go json.Unmarshal supports case-insensitive binding.  However the
			// url params are bound case-sensitive which is inconsistent.  To
			// fix this we must check all of the map values in a
			// case-insensitive search.
			for k, v := range data {
				if strings.EqualFold(k, inputFieldName) {
					inputValue = v
					exists = true
					break
				}
			}
		}

		if !exists {
			if opts.requireAllFields || isRequiredTag(typeField.Tag) {
				missing = append(missing, inputFieldName)
			}
			continue
		}

		// NOTE: algorithm here is not particularly sophisticated. It probably does not work with absurd types like `**[]*int`
		// but it is smart enough to handle niche cases like `*int`,`*[]string`,`[]*int` .

		// try unmarshalling first, in case we're dealing with an alias to an array type
		if ok, err := unmarshalInputsToField(typeField.Type.Kind(), inputValue, structField); ok {
			if err != nil {
				return err
			}
			continue
		}

		formatTag := typeField.Tag.Get("format")
		if ok, err := unmarshalInputToField(typeField.Type.Kind(), inputValue[0], structField, formatTag); ok {
			if err != nil {
				return err
			}
			continue
		}

		// we could be dealing with pointer to slice `*[]string` so dereference it. There are weird OpenAPI generators
		// that could create struct fields like that.
		if structFieldKind == reflect.Pointer {
			structFieldKind = structField.Elem().Kind()
			structField = structField.Elem()
		}

		if structFieldKind == reflect.Slice {
			sliceOf := structField.Type().Elem().Kind()
			numElems := len(inputValue)
			slice := reflect.MakeSlice(structField.Type(), numElems, numElems)
			for j := 0; j < numElems; j++ {
				if err := setWithProperType(sliceOf, inputValue[j], slice.Index(j)); err != nil {
					return err
				}
			}
			structField.Set(slice)
			continue
		}

		if err := setWithProperType(structFieldKind, inputValue[0], structField); err != nil {
			return err
		}
	}
	if len(missing) > 0 {
		return missingRequiredFieldsError(missing)
	}
	return nil
}

var (
	// NOT supported by bind as you can NOT check easily empty struct being actual file or not
	multipartFileHeaderType = reflect.TypeFor[multipart.FileHeader]()
	// supported by bind as you can check by nil value if file existed or not
	multipartFileHeaderPointerType      = reflect.TypeFor[*multipart.FileHeader]()
	multipartFileHeaderSliceType        = reflect.TypeFor[[]multipart.FileHeader]()
	multipartFileHeaderPointerSliceType = reflect.TypeFor[[]*multipart.FileHeader]()
)

func isFieldMultipartFile(field reflect.Type) (bool, error) {
	switch field {
	case multipartFileHeaderPointerType,
		multipartFileHeaderSliceType,
		multipartFileHeaderPointerSliceType:
		return true, nil
	case multipartFileHeaderType:
		return true, errors.New("binding to multipart.FileHeader struct is not supported, use pointer to struct")
	default:
		return false, nil
	}
}

func setMultipartFileHeaderTypes(structField reflect.Value, inputFieldName string, files map[string][]*multipart.FileHeader) bool {
	fileHeaders := files[inputFieldName]
	if len(fileHeaders) == 0 {
		return false
	}

	result := true
	switch structField.Type() {
	case multipartFileHeaderPointerSliceType:
		structField.Set(reflect.ValueOf(fileHeaders))
	case multipartFileHeaderSliceType:
		headers := make([]multipart.FileHeader, len(fileHeaders))
		for i, fileHeader := range fileHeaders {
			headers[i] = *fileHeader
		}
		structField.Set(reflect.ValueOf(headers))
	case multipartFileHeaderPointerType:
		structField.Set(reflect.ValueOf(fileHeaders[0]))
	default:
		result = false
	}

	return result
}

// bindMultipleUnmarshaler is used by binder to unmarshal multiple values from request at once to
// type implementing this interface. For example request could have multiple query fields `?a=1&a=2&b=test` in that case
// for `a` following slice `["1", "2"] will be passed to unmarshaller.
type bindMultipleUnmarshaler interface {
	UnmarshalParams(params []string) error
}

func unmarshalInputsToField(valueKind reflect.Kind, values []string, field reflect.Value) (bool, error) {
	if valueKind == reflect.Ptr {
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		field = field.Elem()
	}

	fieldIValue := field.Addr().Interface()
	unmarshaler, ok := fieldIValue.(bindMultipleUnmarshaler)
	if !ok {
		return false, nil
	}
	return true, unmarshaler.UnmarshalParams(values)
}

func unmarshalInputToField(valueKind reflect.Kind, val string, field reflect.Value, formatTag string) (bool, error) {
	if valueKind == reflect.Ptr {
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		field = field.Elem()
	}

	fieldIValue := field.Addr().Interface()
	// Handle time.Time with custom format tag
	if formatTag != "" {
		if _, isTime := fieldIValue.(*time.Time); isTime {
			t, err := time.Parse(formatTag, val)
			if err != nil {
				return true, err
			}
			field.Set(reflect.ValueOf(t))
			return true, nil
		}
	}

	switch unmarshaler := fieldIValue.(type) {
	case echo.BindUnmarshaler:
		return true, unmarshaler.UnmarshalParam(val)
	case encoding.TextUnmarshaler:
		return true, unmarshaler.UnmarshalText([]byte(val))
	}

	return false, nil
}

func setWithProperType(valueKind reflect.Kind, val string, structField reflect.Value) error {
	// But also call it here, in case we're dealing with an array of BindUnmarshalers
	// Note: format tag not available in this context, so empty string is passed
	if ok, err := unmarshalInputToField(valueKind, val, structField, ""); ok {
		return err
	}

	switch valueKind {
	case reflect.Ptr:
		return setWithProperType(structField.Elem().Kind(), val, structField.Elem())
	case reflect.Int:
		return setIntField(val, 0, structField)
	case reflect.Int8:
		return setIntField(val, 8, structField)
	case reflect.Int16:
		return setIntField(val, 16, structField)
	case reflect.Int32:
		return setIntField(val, 32, structField)
	case reflect.Int64:
		return setIntField(val, 64, structField)
	case reflect.Uint:
		return setUintField(val, 0, structField)
	case reflect.Uint8:
		return setUintField(val, 8, structField)
	case reflect.Uint16:
		return setUintField(val, 16, structField)
	case reflect.Uint32:
		return setUintField(val, 32, structField)
	case reflect.Uint64:
		return setUintField(val, 64, structField)
	case reflect.Bool:
		return setBoolField(val, structField)
	case reflect.Float32:
		return setFloatField(val, 32, structField)
	case reflect.Float64:
		return setFloatField(val, 64, structField)
	case reflect.String:
		structField.SetString(val)
	default:
		return errors.New("unknown type")
	}
	return nil
}

func setIntField(value string, bitSize int, field reflect.Value) error {
	if value == "" {
		value = "0"
	}
	intVal, err := strconv.ParseInt(value, 10, bitSize)
	if err == nil {
		field.SetInt(intVal)
	}
	return err
}

func setUintField(value string, bitSize int, field reflect.Value) error {
	if value == "" {
		value = "0"
	}
	uintVal, err := strconv.ParseUint(value, 10, bitSize)
	if err == nil {
		field.SetUint(uintVal)
	}
	return err
}

func setBoolField(value string, field reflect.Value) error {
	if value == "" {
		value = "false"
	}
	boolVal, err := strconv.ParseBool(value)
	if err == nil {
		field.SetBool(boolVal)
	}
	return err
}

func setFloatField(value string, bitSize int, field reflect.Value) error {
	if value == "" {
		value = "0.0"
	}
	floatVal, err := strconv.ParseFloat(value, bitSize)
	if err == nil {
		field.SetFloat(floatVal)
	}
	return err
}
