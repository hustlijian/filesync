package comm

import (
	"encoding/json"
)

func PackCheckToken(token string) ([]byte, error) {
	var data []byte
	request := Request{
		Cmd:  CheckToken,
		Data: []byte(token),
	}

	data, err := json.Marshal(request)
	if err != nil {
		return data, err
	}
	return data, nil
}
