package main

import (
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/hex"
	"encoding/json"
	"os"
)

func main() {
	info, err := buildinfo.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	digest := sha256.Sum256(raw)
	if err := json.NewEncoder(os.Stdout).Encode(map[string]any{
		"binary_sha256": hex.EncodeToString(digest[:]), "build_info": info,
	}); err != nil {
		panic(err)
	}
}
