package ref

import (
	"fmt"
	"reflect"
	"strings"

	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"
	"code.xxxxx.cn/platform/galaxy/pkg/util/sets"
)

// Diff compare x and y, return fields of x not present in y, x-y
// keys is used to check struct slice if equal
func Diff(x, y interface{}, keys ...string) (interface{}, error) {

	if !typeIsStruct(x) {
		return nil, fmt.Errorf("interface ref type must be struct")
	}
	if !typeEqual(x, y) {
		return nil, fmt.Errorf("not the same type obj")
	}
	xt := typeRef(x)
	xv := valueRef(x)
	yv := valueRef(y)
	if isNil(x) {
		return nil, nil
	}
	if isNil(y) {
		re := reflect.New(xt)
		re.Elem().Set(xv)
		return re.Interface(), nil
	}
	zv := reflect.New(xt)
	for i := 0; i < xv.NumField(); i++ {
		xf := xv.Field(i)
		yf := yv.Field(i)
		zf := zv.Elem().Field(i)
		if !zf.CanSet() || (isPtr(zf) && isBlank(xf)) {
			continue
		}
		// if is key reserved
		if sets.NewString(keys...).Has(fieldKey(x, i)) && !typeIsStruct(xf.Interface()) &&
			!typeIsSlice(xf.Interface()) && (isBlank(xf) || reflect.DeepEqual(xf.Interface(), yf.Interface())) {
			zf.Set(yf)
			continue
		}

		switch xf.Kind() {
		case reflect.Ptr, reflect.Struct:
			diff, err := Diff(xf.Interface(), yf.Interface(), keys...)
			if err != nil {
				return nil, err
			}
			if zf.Kind() == reflect.Ptr {
				zf.Set(reflect.ValueOf(diff))
			} else {
				zf.Set(reflect.ValueOf(diff).Elem())
			}
		case reflect.Slice:
			// merge slice by field id
			diff, err := sliceDiff(xf.Interface(), yf.Interface(), keys...)
			if err != nil {
				return nil, err
			}
			if diff != nil {
				zf.Set(reflect.ValueOf(diff))
			}
		default:
			if !reflect.DeepEqual(xf.Interface(), yf.Interface()) {
				zf.Set(xf)
			}
		}
	}

	return zv.Interface(), nil
}

func sliceDiff(x, y interface{}, keys ...string) (interface{}, error) {
	if !typeEqual(x, y) {
		return nil, fmt.Errorf("not the same type obj")
	}
	xt, err := sliceElemType(x)
	if err != nil {
		return nil, err
	}
	if _, err := sliceElemType(y); err != nil {
		return nil, err
	}
	if xt.Kind() == reflect.Slice {
		return nil, fmt.Errorf("not support slice in slice")
	}
	if len(keys) == 0 && reflect.DeepEqual(x, y) {
		return nil, nil
	}

	yv := reflect.ValueOf(y)
	xv := reflect.ValueOf(x)
	z := reflect.New(reflect.TypeOf(x)).Elem()
	z = reflect.AppendSlice(z, reflect.ValueOf(x))

	zidx := 0
	for i := 0; i < xv.Len(); i++ {
		existsIndex := -1
		for j := 0; j < yv.Len(); j++ {
			if z.Index(zidx).CanSet() && equalsByKeys(xv.Index(i).Interface(), yv.Index(j).Interface(), keys...) {
				diff, err := Diff(xv.Index(i).Interface(), yv.Index(j).Interface(), keys...)
				if err != nil {
					return nil, err
				}
				if diff != nil {
					if z.Index(zidx).Kind() == reflect.Ptr {
						z.Index(zidx).Set(reflect.ValueOf(diff))
					} else {
						z.Index(zidx).Set(reflect.ValueOf(diff).Elem())
					}
				}
				break
			} else if reflect.DeepEqual(xv.Index(i).Interface(), yv.Index(j).Interface()) {
				existsIndex = zidx
				break
			}
		}
		if existsIndex > -1 {
			// delete deep equals elem
			z = reflect.AppendSlice(z.Slice(0, existsIndex), z.Slice(existsIndex+1, z.Len()))
			zidx--
		}
		zidx++
	}
	return z.Interface(), nil
}

func sliceUnion(x, y interface{}, keys ...string) (interface{}, error) {
	if !typeEqual(x, y) {
		return nil, fmt.Errorf("not the same type obj")
	}
	xt, err := sliceElemType(x)
	if err != nil {
		return nil, err
	}
	if _, err := sliceElemType(y); err != nil {
		return nil, err
	}
	if xt.Kind() == reflect.Slice {
		return nil, fmt.Errorf("not support slice in slice")
	}
	if reflect.DeepEqual(x, y) {
		return x, nil
	}

	xv := reflect.ValueOf(x)
	yv := reflect.ValueOf(y)
	z := reflect.New(reflect.TypeOf(x)).Elem()
	z = reflect.AppendSlice(z, xv)
	for j := 0; j < yv.Len(); j++ {
		exists := false
		for i := 0; i < xv.Len(); i++ {
			if z.Index(i).CanSet() && equalsByKeys(xv.Index(i).Interface(), yv.Index(j).Interface(), keys...) {
				union, err := Union(xv.Index(i).Interface(), yv.Index(j).Interface(), keys...)
				if err != nil {
					return nil, err
				}
				if z.Index(i).Kind() == reflect.Ptr {
					z.Index(i).Set(reflect.ValueOf(union))
				} else {
					z.Index(i).Set(reflect.ValueOf(union).Elem())
				}
				exists = true
				break
			}
		}
		if !exists {
			z = reflect.Append(z, yv.Index(j))
		}
	}
	return z.Interface(), nil
}

// only struct if equals by any keys
func equalsByKeys(x, y interface{}, keys ...string) bool {
	if len(keys) == 0 {
		return false
	}
	if isNil(x) || isNil(y) {
		return false
	}
	if !typeEqual(x, y) {
		return false
	}
	if !typeIsStruct(x) {
		return false
	}
	xv := valueRef(x)
	yv := valueRef(y)

	for _, key := range keys {
		k := strings.Split(key, ".")[len(strings.Split(key, "."))-1]
		xf := xv.FieldByName(k)
		yf := yv.FieldByName(k)
		if xf.IsValid() {
			if reflect.DeepEqual(xf.Interface(), yf.Interface()) {
				return true
			}
			if typeIsStruct(xf.Interface()) &&
				equalsByKeys(xf.Interface(), yf.Interface(), keys...) {
				return true
			}
		}
	}
	return false
}

// sliceElemType return slice elem type
func sliceElemType(sl interface{}) (reflect.Type, error) {
	if reflect.TypeOf(sl).Kind() != reflect.Slice {
		alog.Errorf("[%v] not a slice but %T", sl, sl)
		return nil, fmt.Errorf("[%v] not a slice but %T", sl, sl)
	}
	return reflect.TypeOf(sl).Elem(), nil
}

// Copy return a copy obj of x
func Copy(x interface{}) (interface{}, error) {
	if isNil(x) {
		return nil, nil
	}

	if !typeIsStruct(x) {
		return nil, fmt.Errorf("only support copy struct")
	}

	xt := typeRef(x)
	xv := valueRef(x)

	y := reflect.New(xt)
	yv := y.Elem()
	for i := 0; i < xv.NumField(); i++ {
		value := xv.Field(i)
		dvalue := yv.Field(i)
		if !dvalue.IsValid() || !dvalue.CanSet() {
			alog.Errorf("%T value %v %v invalid or can't be set", x, dvalue.Kind(), dvalue)
			continue
		}
		dvalue.Set(value)
	}
	return y.Interface(), nil
}

// Union compare x and y, overwrite fields of x by none zero fields of y, x+y
func Union(x, y interface{}, keys ...string) (interface{}, error) {
	if reflect.DeepEqual(x, y) {
		return Copy(x)
	}
	if isNil(x) {
		return Copy(y)
	}
	if isNil(y) {
		return Copy(x)
	}
	if !typeEqual(x, y) {
		return nil, fmt.Errorf("not the same type obj")
	}

	cp, err := Copy(x)
	if err != nil {
		return nil, err
	}

	cpv := valueRef(cp)
	yv := valueRef(y)
	for i := 0; i < cpv.NumField(); i++ {
		cpf := cpv.Field(i)
		yvf := yv.Field(i)
		if !isBlank(yvf) && cpf.CanSet() {
			switch yvf.Kind() {
			case reflect.Ptr, reflect.Struct:
				union, err := Union(cpf.Interface(), yvf.Interface(), keys...)
				if err != nil {
					return nil, err
				}
				if cpf.Kind() == reflect.Ptr {
					cpf.Set(reflect.ValueOf(union))
				} else {
					cpf.Set(reflect.ValueOf(union).Elem())
				}
			case reflect.Slice:
				union, err := sliceUnion(cpf.Interface(), yvf.Interface(), keys...)
				if err != nil {
					return nil, err
				}
				cpf.Set(reflect.ValueOf(union))
			default:
				cpf.Set(yvf)
			}
		}
	}
	return cp, nil
}
func isPtr(value reflect.Value) bool {
	return value.Kind() == reflect.Ptr
}
func isBlank(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.String:
		return value.Len() == 0
	case reflect.Bool:
		return !value.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return value.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return value.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return value.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return value.IsNil()
	case reflect.Slice:
		return value.IsNil() || value.Len() == 0
	}
	return reflect.DeepEqual(value.Interface(), reflect.Zero(value.Type()).Interface())
}

func isNil(x interface{}) bool {
	value := reflect.ValueOf(x)
	return value.Kind() == reflect.Ptr && value.IsNil()
}

func typeEqual(x, y interface{}) bool {
	xt := reflect.TypeOf(x)
	yt := reflect.TypeOf(y)
	if xt.Kind() == reflect.Ptr {
		xt = xt.Elem()
	}
	if yt.Kind() == reflect.Ptr {
		yt = yt.Elem()
	}
	return xt == yt
}

func typeIsStruct(x interface{}) bool {
	xt := typeRef(x)
	return xt.Kind() == reflect.Struct
}

func typeIsSlice(x interface{}) bool {
	xt := typeRef(x)
	return xt.Kind() == reflect.Slice
}

func typeRef(x interface{}) reflect.Type {
	xt := reflect.TypeOf(x)
	if xt.Kind() == reflect.Ptr {
		xt = xt.Elem()
	}
	return xt
}

func valueRef(x interface{}) reflect.Value {
	xv := reflect.ValueOf(x)
	if xv.Kind() == reflect.Ptr {
		xv = reflect.ValueOf(x).Elem()
	} else {
		xv = reflect.ValueOf(x)
	}
	return xv
}

func fieldKey(x interface{}, fieldIdx int) string {
	names := strings.Split(fmt.Sprintf("%T", x), ".")
	return fmt.Sprintf("%s.%s", names[len(names)-1], typeRef(x).Field(fieldIdx).Name)
}
