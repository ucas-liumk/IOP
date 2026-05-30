package dictionary

import "encoding/json"

type jsonHelper struct{}

func (jsonHelper) marshal(v any) ([]byte, error)         { return json.Marshal(v) }
func (jsonHelper) unmarshal(b []byte, v any) error       { return json.Unmarshal(b, v) }

var jsonImpl = jsonHelper{}
