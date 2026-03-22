package monitor

import (
	"encoding/json"
	"strconv"
)

// jsonInt64 handles JSON fields that may be either a number or a string.
type jsonInt64 int64

func (j *jsonInt64) UnmarshalJSON(data []byte) error {
	var n json.Number
	if err := json.Unmarshal(data, &n); err != nil {
		return err
	}
	i, err := strconv.ParseInt(n.String(), 10, 64)
	if err != nil {
		return err
	}
	*j = jsonInt64(i)
	return nil
}

func (j jsonInt64) MarshalJSON() ([]byte, error) {
	return json.Marshal(int64(j))
}

func (j jsonInt64) Int64() int64 {
	return int64(j)
}
