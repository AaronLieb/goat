package ioutil

import (
	"io"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	json "github.com/neilotoole/jsoncolor"
)

type M map[string]interface{}

func Flatten(results [][]types.ResultField) []M {
	var flatResults []M
	for _, log := range results {
		flatLog := make(M)
		for _, entry := range log {
			if *entry.Field == "@ptr" {
				continue
			}
			if *entry.Field == "@message" {
				var dest map[string]*json.RawMessage
				json.Unmarshal([]byte(*entry.Value), &dest)
				flatLog[*entry.Field] = dest
				continue
			}
			flatLog[*entry.Field] = *entry.Value
		}
		flatResults = append(flatResults, flatLog)
	}
	return flatResults
}

func GetEncoder(buf *io.Writer, isColor bool) *json.Encoder {
	var enc *json.Encoder
	if isColor {
		enc = json.NewEncoder(*buf)
		enc.SetColors(json.DefaultColors())
	} else {
		enc = json.NewEncoder(*buf)
	}
	enc.SetIndent("", "  ")
	return enc
}
