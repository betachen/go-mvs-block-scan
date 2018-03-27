package config

import (
	"encoding/json"
	"github.com/inconshreveable/log15"
	"io/ioutil"
)

var log = log15.New()

var content map[string]map[string]string

func LoadFile(filepath string) bool {
	file, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.Info("LoadFile error:", err)
		return false
	}
	err = json.Unmarshal(file, &content)
	if err != nil {
		log.Info("LoadFile error:", err)
		return false
	}
	return true
}

func Get(table, key string) string {
	obj, ok := content[table]
	if !ok {
		log.Info("Open config file failed:", table)
		return ""
	}
	val, ok := obj[key]
	if !ok {
		log.Info("Open config file failed:", key)
		return ""
	}
	return val
}

func Set(table, key, val string) {
	obj, ok := content[key]
	if ok {
		obj[key] = val
	}
}

func Global(key string) string {
	val, ok := content["global"][key]
	if !ok {
		return ""
	}
	return val
}
