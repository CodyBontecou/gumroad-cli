package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

func PrintNDJSON(w io.Writer, writeItems func(func(any) error) error) error {
	if writeItems == nil {
		return nil
	}
	return writeItems(func(item any) error {
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(item); err != nil {
			return fmt.Errorf("could not encode NDJSON record: %w", err)
		}
		if _, err := w.Write(buf.Bytes()); err != nil {
			return err
		}
		return nil
	})
}
