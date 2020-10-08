package main

import (
	"fmt"
	"os"
)

var (
	FreyaKey        = os.Getenv("FREYA") // nolint:gochecknoglobals
	MassDNSChecksum = "unset"            // nolint:gochecknoglobals
)

func main() {
	fmt.Println(MassDNSChecksum)
	fmt.Println(FreyaKey)
}
