package utils

import (
	"reflect"
)

// IndexOf is a function that can be used
// to retrieve the index of an element in 
// a slice.
// Returns: The index of the element or -1
func IndexOf(sl interface{}, el interface{}) int {
	slice := reflect.ValueOf(sl)
	if slice.Kind() != reflect.Slice {
		return -1
	}

	for i := 0; i < slice.Len(); i++ {
		if slice.Index(i).Interface() == el {
			return i
		}
	}

	return -1
}