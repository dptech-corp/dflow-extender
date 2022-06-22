package util

import (
	"log"
)

type Config map[string]interface{}

func (c Config) GetValue(k string) interface{} {
	v, ok := c[k]
	if !ok {
		log.Fatal("Config '" + k + "' not found")
	}
	return v
}
