package virtualbox

import (
	"fmt"
	"log"
)

// errLogf is an abstraction function which allows you to both log and return an error
func errLogf(format string, args ...interface{}) error {
	// TODO: Consider call depth if we add line logging to the errors
	e := fmt.Errorf("[ERROR] "+format, args...)
	log.Println(e)
	return e
}

func getMapValueAsString(m map[string]interface{}, key string) (value string, err error) {
	valueI, ok := m[key]
	if !ok {
		return "", nil
	}
	value, ok = valueI.(string)
	if !ok {
		return "", fmt.Errorf("could not convert map element to string: map[%s]=%#v", key, valueI)
	}

	return value, nil
}
