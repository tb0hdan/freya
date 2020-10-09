package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
)

var (
	FreyaKey        = os.Getenv("FREYA") // nolint:gochecknoglobals
	MassDNSChecksum = "unset"            // nolint:gochecknoglobals
)

func SHA256Sum(fname string) (sum string, err error) {
	f, err := os.Open(fname)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func main() {
	binarySum, err := SHA256Sum("/massdns")
	if err != nil {
		panic(err)
	}

	if binarySum != MassDNSChecksum {
		log.Fatalf("massdns checksum mismatch: `%s` `%s`", binarySum, MassDNSChecksum)
	}

	if len(FreyaKey) == 0 {
		log.Fatalf("cannot run without FREYA key")
	}

	fmt.Println(FreyaKey)
}
