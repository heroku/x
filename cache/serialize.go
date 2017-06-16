package cache

import "errors"

var errNotString = errors.New("StringEncoder: Unable to Encode non-string")

// StringEncoder ...
func StringEncoder(v interface{}) ([]byte, error) {
	str, ok := v.(string)
	if !ok {
		return nil, errNotString
	}
	return []byte(str), nil
}

// StringDecoder ...
func StringDecoder(buf []byte) (interface{}, error) {
	return string(buf), nil
}
