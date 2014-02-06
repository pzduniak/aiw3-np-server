package structs

import (
	"strings"
)

func StripPort(host string) string {
	return strings.Split(host, ":")[0]
}
