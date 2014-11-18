package main

import (
	"github.com/eliwjones/thebox/adapter/tdameritrade"
	"github.com/eliwjones/thebox/util/funcs"

	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"
)

func main() {
	id, pass, sid, jsess, _ := funcs.GetConfig()

	tda := tdameritrade.New(id, pass, sid, jsess)
	if tda.JsessionID != jsess {
		funcs.UpdateConfig(id, pass, sid, tda.JsessionID)
	}
	options, err := tda.GetOptions("INTC")

	if err != nil {
		fmt.Sprintf("Got err: %s", err)
	}
	for _, option := range options {
		now := time.Now()
		limit := now.AddDate(0, 0, 22).Format("20060102")
		if option.Expiration > limit {
			continue
		}
		stuff, err := json.Marshal(option)
		if err != nil {
			fmt.Sprintf("Got err: %s", err)
		}
		key := fmt.Sprintf("%s_%s_%s_%d", option.Underlying, option.Expiration, option.Type, option.Strike)
		lazyWriteFile("data", key, stuff)
	}
}

func lazyWriteFile(folderName string, fileName string, data []byte) error {
	err := ioutil.WriteFile(folderName+"/"+fileName, data, 0777)
	if err != nil {
		os.MkdirAll(folderName, 0777)
		err = ioutil.WriteFile(folderName+"/"+fileName, data, 0777)
	}
	if err != nil {
		fmt.Printf("[LazyWriteFile] Could not WriteFile: %s\nErr: %s\n", folderName+"/"+fileName, err)
	}
	return err
}
