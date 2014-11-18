package funcs

import (
	"fmt"
	"io/ioutil"
	"strings"
	"time"
)

var MS = func(time time.Time) int64 {
	return time.UnixNano() / 1000000
}

var Now = func() time.Time { return time.Now() }

func GetConfig() (string, string, string, string, error) {
	b, err := ioutil.ReadFile("config")
	if err != nil {
		return "", "", "", "", err
	}
	lines := strings.Split(string(b), "\n")
	return lines[0], lines[1], lines[2], lines[3], nil
}

func UpdateConfig(id string, pass string, sid string, jsess string) error {
	f := []byte(fmt.Sprintf("%s\n%s\n%s\n%s", id, pass, sid, jsess))
	err := ioutil.WriteFile("config", f, 0777)
	return err
}
