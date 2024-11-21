package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/snksoft/crc"
)

func main() {
	// Used for default file in cmd line args.
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
		return
	}
	defaultFilePath := filepath.Join(home, "Pictures", "smiley.png")

	// cl-args for png file path
	var png string
	flag.StringVar(&png, "png", defaultFilePath, "png file to supply")

	// NOTE:
	flag.Parse()

	// Open the png
	file, err := os.Open(png)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer file.Close()

	log.Printf("Successfully opened %s\n", png)

	decoder := NewPngDecoder()
	if _, err := decoder.IsPng(file); err != nil {
		log.Fatal(err)
	}

	err = decoder.ParseChunkStream(file)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("PNG file parsed successfully!")
}

// PngDecoder represents the process of reconstructing the reference image from a
// PNG datastream and generating a delivered image.
type PngDecoder struct {
	Chunks []Chunk
}

// NewPngDecoder creates a new PngDecoder
func NewPngDecoder() *PngDecoder {
	return &PngDecoder{
		Chunks: make([]Chunk, 0),
	}
}

// ParseChunkStream parses (decodes) a PNG datastream based on the sequence of
// chunks read in.
// NOTE: This method assumes a valid PNG file has already been passed in.
func (p *PngDecoder) ParseChunkStream(file *os.File) error {
Loop:
	for {
		chunk, err := p.readChunk(file)
		switch {
		case err != nil:
			return err
		case chunk.Type.slug == ChunkIEND.slug:
			log.Println("Reached IEND")
			break Loop
		}
	}
	return nil
}

// isPng determines if a file is a PNG file by examining the PNG signature.
func (p *PngDecoder) IsPng(file *os.File) (bool, error) {
	// 137 80 78 71 13 10 26 10
	const pngSignatureHex = "\x89\x50\x4E\x47\x0D\x0A\x1A\x0A"

	// First 8 bytes of the PNG datastream should be the same as the const above.
	signature := make([]byte, 8)
	n, err := file.Read(signature)
	switch {
	case err != nil:
		return false, err
	case n != len(pngSignatureHex):
		return false, fmt.Errorf("n: %d and pngSignatureHex length: %d are mismatched", n, len(pngSignatureHex))
	case !bytes.Equal(signature, []byte(pngSignatureHex)):
		return false, fmt.Errorf("signature mismatch: got %x, expected %x", signature, pngSignatureHex)
	}

	log.Println("Successfully validated PNG signature!")

	return true, nil
}

// readChunk is a helper to read a single chunk of PNG data.
func (p *PngDecoder) readChunk(file *os.File) (*Chunk, error) {
	// Below is visually what a chunk in the PNG datastream looks like.
	//  +------------+ +------------+ +------------+ +-------+
	//  |   LENGTH   | | CHUNK TYPE | | CHUNK DATA | |  CRC  |
	//  +------------+ +------------+ +------------+ +-------+

	// Step 1: Read 4 integer bytes, the length of the chunk data field.
	var length uint32
	err := binary.Read(file, binary.BigEndian, &length)
	if err != nil {
		return nil, fmt.Errorf("binary.Read failed: %d", length)
	}
	log.Printf("Successfully read the length: %d\n", length)

	// Step 2: Read 4 bytes of chunk type data.
	readType := make([]byte, 4)
	if _, err := file.Read(readType); err != nil {
		return nil, fmt.Errorf("io.Read failed to read the chunkType")
	}
	// Convert the first four bytes to a string
	chunkBuffer := string(readType)

	// Get the chunk type
	chunkType, err := FromString(chunkBuffer)
	if err != nil {
		return nil, fmt.Errorf("unknown chunk type: %s", chunkType)
	}
	log.Printf("chunkType: %v\n", chunkType)

	// Step 3: Read the chunk data according to type
	chunkData := make([]byte, length)

	n, err := file.Read(chunkData)
	switch {
	case n != int(length):
		return nil, fmt.Errorf("n: %d and length: %d are mismatched for chunk data", n, length)
	case err != nil:
		return nil, fmt.Errorf("failed to read chunk data")
	}

	// Step 4a: Read in the crc chunk. It is a 4-byte integer, but I need it in bytes to compute the CRC.
	var storedCRCChunk uint32
	err = binary.Read(file, binary.BigEndian, &storedCRCChunk)
	if err != nil {
		return nil, fmt.Errorf("binary.Read failed: %d", storedCRCChunk)
	}
	log.Printf("storedCRCChunk: %d\n", storedCRCChunk)
	// TODO: Step 4b: Validate the crc chunk
	// Essentially, we need to create a CRC object, compute it using the chunk type + chunk data, and compare
	// it with the crc we read in.

	// The four-byte CRC is calculated on the preceding bytes in the chunk: chunk type + chunk data.
	precedingBytes := append(readType, chunkData...)

	// Create crc object
	crc32 := crc.CRC32
	// Compute the CRC
	computedCRC := crc.CalculateCRC(crc32, precedingBytes)
	// Validate the computed CRC versus the CRC stored in the PNG datastream.
	if uint32(computedCRC) == storedCRCChunk {
		log.Printf("checksums match for CRC validation: stored %d, calculated %d\n", storedCRCChunk, computedCRC)
	} else {
		return nil, fmt.Errorf("checksums failed for CRC validation: stored %d, calculated %d", storedCRCChunk, computedCRC)
	}

	// IHDR chunk is the FIRST chunk in the PNG datastream.
	type IHDR struct {
		width             uint32
		height            uint32
		bitDepth          uint8
		colorType         uint8
		compressionMethod uint8
		filterMethod      uint8
		interlaceMethod   uint8
	}

	// Extract the chunk data and store (or ignore) for relevant chunk type.
	switch chunkType {
	case ChunkIHDR:
		// TODO: fill in IHDR chunk values
		ihdr := IHDR{
			width:             binary.BigEndian.Uint32(chunkData[0:4]),
			height:            binary.BigEndian.Uint32(chunkData[4:8]),
			bitDepth:          chunkData[8],
			colorType:         chunkData[9],
			compressionMethod: chunkData[10],
			filterMethod:      chunkData[11],
			interlaceMethod:   chunkData[12],
		}
		log.Printf("Parsed IHDR: %+v\n", ihdr)
	case ChunkIEND:
		log.Println("IEND. Done!")
	default:
		fmt.Printf("Skipping chunk type: %s\n", chunkType)
	}

	return &Chunk{
		Length: length,
		Type:   ChunkType(chunkType),
		Data:   chunkData,
		Crc:    uint32(computedCRC),
	}, nil
}

// Chunk defines the chunk layout as specified by PNG datastream structure.
type Chunk struct {
	Length uint32    // A four-byte unsigned integer giving the number of bytes in the chunk's data field.
	Type   ChunkType // A sequence of four bytes defining the chunk type.
	Data   []byte    // The data bytes of the relevant chunk type; can be zero length.
	Crc    uint32    // A four-byte CRC (Cyclic Redundancy Code) calculated on the preceding bytes in the chunk.
	// Includes chunk type and data, but NOT length.
}

// isCritical determines if a chunk is a Ancillary or Critical type.
func (c *Chunk) isCritical() bool {
	return c.Type.slug[0] >= 'A' && c.Type.slug[0] <= 'Z'
}

type ChunkType struct {
	slug string
}

func (c ChunkType) String() string {
	return c.slug
}

func FromString(s string) (ChunkType, error) {
	switch s {
	case ChunkIHDR.slug:
		return ChunkIHDR, nil
	case ChunkPLTE.slug:
		return ChunkPLTE, nil
	case ChunkIDAT.slug:
		return ChunkIDAT, nil
	case ChunkIEND.slug:
		return ChunkIEND, nil
	case ChunkcHRM.slug:
		return ChunkcHRM, nil
	case ChunkgAMA.slug:
		return ChunkgAMA, nil
	case ChunkiCCP.slug:
		return ChunkiCCP, nil
	case ChunksBIT.slug:
		return ChunksBIT, nil
	case ChunksRGB.slug:
		return ChunksRGB, nil
	case ChunkbKGD.slug:
		return ChunkbKGD, nil
	case ChunkhIST.slug:
		return ChunkhIST, nil
	case ChunktRNS.slug:
		return ChunktRNS, nil
	case ChunkpHYs.slug:
		return ChunkpHYs, nil
	case ChunksPLT.slug:
		return ChunksPLT, nil
	case ChunktIME.slug:
		return ChunktIME, nil
	case ChunkiTXt.slug:
		return ChunkiTXt, nil
	case ChunktEXt.slug:
		return ChunktEXt, nil
	case ChunkzTXt.slug:
		return ChunkzTXt, nil
	}

	return Unknown, errors.New("unknown chunk type")
}

var (
	Unknown = ChunkType{""}

	// NOTE: Critical chunks
	ChunkIHDR = ChunkType{"IHDR"}
	ChunkPLTE = ChunkType{"PLTE"}
	ChunkIDAT = ChunkType{"IDAT"}
	ChunkIEND = ChunkType{"IEND"}

	// NOTE:  Ancillary chunks
	ChunkcHRM = ChunkType{"cHRM"}
	ChunkgAMA = ChunkType{"gAMA"}
	ChunkiCCP = ChunkType{"iCCP"}
	ChunksBIT = ChunkType{"sBIT"}
	ChunksRGB = ChunkType{"sRGB"}
	ChunkbKGD = ChunkType{"bKGD"}
	ChunkhIST = ChunkType{"hIST"}
	ChunktRNS = ChunkType{"tRNS"}
	ChunkpHYs = ChunkType{"pHYs"}
	ChunksPLT = ChunkType{"sPLT"}
	ChunktIME = ChunkType{"tIME"}
	ChunkiTXt = ChunkType{"iTXt"}
	ChunktEXt = ChunkType{"tEXt"}
	ChunkzTXt = ChunkType{"zTXt"}
)
