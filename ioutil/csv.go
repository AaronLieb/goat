package ioutil

import (
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

func PrintCsv(buf *io.Writer, data [][]types.ResultField) {

	if len(data) == 0 {
		return
	}

	// Print the headers
	for i, entry := range data[0] {
		if i != 0 {
			fmt.Fprint(*buf, ", ")
		}
		fmt.Fprint(*buf, *entry.Field)
	}
	fmt.Fprintln(*buf)

	// Print the values
	for _, log := range data {
		for i, entry := range log {
			if i != 0 {
				fmt.Fprint(*buf, ", ")
			}
			fmt.Fprint(*buf, *entry.Value)
		}
		fmt.Fprintln(*buf)
	}
}
