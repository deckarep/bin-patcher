package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
)

// PatchDefinition is a definition of one or more file patch sequences.
type PatchDefinition []struct {
	InputFile  string `json:"input-file"`
	OutputFile string `json:"output-file"`
	Sequence   []struct {
		Desc       string `json:"desc"`
		Settings   string `json:"settings"`
		Transition struct {
			Signature string `json:"signature"`
			Patch     string `json:"patch"`
		} `json:"transition"`
	} `json:"sequence"`
}

func main() {
	b, err := ioutil.ReadFile("patch-seq.def")
	if err != nil {
		log.Fatal("failed to read patch sequence file with err:", err)
	}

	var patchDef PatchDefinition
	err = json.Unmarshal(b, &patchDef)
	if err != nil {
		log.Fatal("failed to unmarshal patch definition with err:", err)
	}

	for _, fileSeq := range patchDef {
		f, err := ioutil.ReadFile(fileSeq.InputFile)
		if err != nil {
			log.Fatal("failed to read file defined in file patch sequence with err:", err)
		}
		for i, seq := range fileSeq.Sequence {
			// Ensure in-place editing only.
			if len(seq.Transition.Signature) != len(seq.Transition.Patch) {
				fmt.Printf("Sequence num: %d, filename: %q\n", i, fileSeq.InputFile)
				fmt.Printf("  Signature byte len: %d\n", len(seq.Transition.Signature))
				fmt.Printf("  Patch byte len: %d\n", len(seq.Transition.Patch))
				log.Fatal("Signature and Patch must match in byte size: (in-place changes only)")
			}

			decodedString := decodeHexString(seq.Transition.Signature)
			fmt.Printf("before (%d): %s\n", i, hex.EncodeToString(f))
			if offset, found := identifySignatureOffset(f, decodedString); found {
				decodedPatch := decodeHexString(seq.Transition.Patch)
				applyPatch(f, offset, decodedPatch)

				fmt.Printf("after  (%d): %s\n", i, hex.EncodeToString(f))
			} else {
				log.Fatal("patch transition not applied because: Signature Not Found!")
			}
		}

		// If an output file is specified than dump the final there.
		if len(fileSeq.OutputFile) > 0 {
			err := ioutil.WriteFile(fileSeq.OutputFile, f, 0666)
			if err != nil {
				log.Fatalf("failed to output file to: %q", fileSeq.OutputFile)
			}
		}
	}
}

func decodeHexString(s string) []byte {
	decoded, err := hex.DecodeString(s)
	if err != nil {
		log.Fatal("failed to decode hex string with err:", err)
	}
	return decoded
}

func identifySignatureOffset(f []byte, signature []byte) (int, bool) {
	var offsetIndex int
	var offsetIdentified int
	var matchCount int
	for _, b := range f {
		if b == signature[0] {
			//fmt.Println("first byte found at offset: ", offsetIndex)
			remaining := f[offsetIndex : offsetIndex+len(signature)]
			//fmt.Println("sig: ", signature, "rem: ", remaining)
			if bytes.Equal(signature, remaining) {
				offsetIdentified = offsetIndex
				matchCount++
			}
		}
		offsetIndex++
	}

	if matchCount == 0 {
		return -1, false
	}

	// This is just to make sure we target the specific location...
	// Alternatively we make the tool the user to specify the offset manually, along with the bytes to overwrite.
	if matchCount > 1 {
		log.Fatal("Multiple matches occurred: Signature is not specific enough!!!")
	}

	return offsetIdentified, true
}

func applyPatch(f []byte, offset int, patch []byte) {
	for i, b := range patch {
		f[offset+i] = b
	}
}
