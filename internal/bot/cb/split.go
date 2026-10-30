package cb

import "strings"

func Split(data string) (head, rest string) {
	i := strings.IndexByte(data, '-')
	if i < 0 {
		return data, ""
	}
	return data[:i], data[i+1:]
}
