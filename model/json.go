package model

import (
	"database/sql/driver"
	"encoding/json"
	"github.com/pkg/errors"
)

type JSON map[string]interface{}

func (a JSON) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a *JSON) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &a)
}

func JSONMap(document map[string]interface{}, key string) (map[string]interface{}, bool) {
	if value, ok := document[key]; ok {
		if mapVal, isMap := value.(map[string]interface{}); isMap {
			return mapVal, true
		}
	}
	return nil, false
}

func JSONString(document map[string]interface{}, key string) (string, bool) {
	if value, ok := document[key]; ok {
		if strValue, isString := value.(string); isString {
			return strValue, true
		}
	}
	return "", false
}

func JSONStrings(document map[string]interface{}, key string) ([]string, bool) {
	var results []string
	value, ok := document[key]
	if !ok {
		return nil, false
	}
	switch v := value.(type) {
	case string:
		results = append(results, v)
	case []interface{}:
		for _, el := range v {
			if strValue, isString := el.(string); isString {
				results = append(results, strValue)
			}
		}
	}
	return results, false
}

func StringsContainsString(things []string, value string) bool {
	for _, thing := range things {
		if thing == value {
			return true
		}
	}
	return false
}
