package structure

import (
	"errors"
	"reflect"
)

// BlankStructure resets the fields of a struct to default values, where strings become "string",
// numeric types are zeroed, slices/arrays become empty, and maps are zeroed.
// For nested structs, it recursively applies the same rules.
//
// The function works with both struct values and pointers to structs. It preserves
// the structure of nested objects while resetting their values.
//
// Supported types:
//   - Strings (set to "string")
//   - Integers (all sizes, set to 0)
//   - Floats (all sizes, set to 0)
//   - Booleans (set to false)
//   - Complex numbers (set to 0+0i)
//   - Maps (set to nil)
//   - Slices (set to empty slice)
//   - Arrays (zeroed)
//   - Structs (recursively processed)
//   - Pointers to structs (recursively processed if non-nil)
//
// Parameters:
//   - v: any - the struct or pointer to struct to be processed
//
// Returns:
//   - error - returns nil on success, or an error if:
//   - the input is nil
//   - the input is not a struct or pointer to struct
//   - encounters an unhandled type
//
// Example:
//
//	type Person struct {
//	    name    string
//	    Age     int
//	    Address *struct {
//	        Street string
//	        City   string
//	    }
//	}
//
//	p := Person{
//	    name: "John",
//	    Age:  30,
//	    Address: &struct {
//	        Street string
//	        City   string
//	    }{
//	        Street: "123 Main St",
//	        City:   "Anytown",
//	    },
//	}
//
//	err := BlankStructure(&p)
//	// After: p.name = "string", p.Age = 0, p.Address.Street = "string", p.Address.City = "string"
func BlankStructure(v any) error {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return errors.New("nil pointer")
		}
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return errors.New("not a struct")
	}

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		if !field.CanSet() {
			continue
		}

		switch field.Kind() {
		case reflect.Ptr:
			if !field.IsNil() && field.Elem().Kind() == reflect.Struct {
				if err := BlankStructure(field.Interface()); err != nil {
					return err
				}
			}
		case reflect.Struct:
			if err := BlankStructure(field.Addr().Interface()); err != nil {
				return err
			}
		case reflect.Map:
			field.Set(reflect.Zero(field.Type()))
		case reflect.Slice, reflect.Array:
			field.Set(reflect.MakeSlice(field.Type(), 0, 0))
		case reflect.String:
			field.SetString("string")
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			field.Set(reflect.Zero(field.Type()))
		case reflect.Float32, reflect.Float64:
			field.Set(reflect.Zero(field.Type()))
		case reflect.Bool:
			field.Set(reflect.Zero(field.Type()))
		case reflect.Complex64, reflect.Complex128:
			field.Set(reflect.Zero(field.Type()))
		default:
			return errors.New("unhandled default case")
		}
	}
	return nil
}
