package ut

import (
	"fmt"
	"os"
	"strings"
)

func SaveSliceToFiles(filePath string, values []interface{}, headers ...string) error {
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	if len(headers) > 0 {
		h := strings.Join(headers, ",")
		fmt.Fprintln(f, h)
	}
	for _, value := range values {
		fmt.Fprintln(f, value) // print values to f, one per line
	}
	return nil
}
