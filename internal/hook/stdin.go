package hook

import (
	"encoding/json"
	"io"
	"os"
)

func decodeJSON(r io.Reader, v any) {
	if isInteractiveStdin(r) {
		return
	}
	_ = json.NewDecoder(r).Decode(v)
}

func isInteractiveStdin(r io.Reader) bool {
	file, ok := r.(*os.File)
	if !ok {
		return false
	}
	if file != os.Stdin {
		return false
	}
	st, err := file.Stat()
	if err != nil {
		return false
	}
	return st.Mode()&os.ModeCharDevice != 0
}
