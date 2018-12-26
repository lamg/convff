package main

import (
	"encoding/json"
	"testing"
)

type codec struct {
	Codec_Name      string `json: "codec_name"`
	Codec_Long_Name string `json: "codec_long_name"`
}

func TestJSON(t *testing.T) {
	str := `{
		"codec_name": "aac",
		"bla": "coco",
    "codec_long_name": "AAC (Advanced Audio Coding)"
  }`
	c := new(codec)
	e := json.Unmarshal([]byte(str), c)
	if e == nil {
		t.Logf("%v", c)
	} else {
		t.Log(e)
	}
}
