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

	decoder.readChunk(file)

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
	p.readChunk(file)
	return nil
}

// isPng determines if a file is a PNG file by examining the PNG signature.
func (p *PngDecoder) IsPng(file *os.File) (bool, error) {
	// 137 80 78 71 13 10 26 10
	const pngSignatureHex = "\x89\x50\x4E\x47\x0D\x0A\x1A\x0A"

	// First 8 bytes of the PNG datastream should be the same as the const above.
	signature := make([]byte, 8)
	if _, err := file.Read(signature); err != nil {
		return false, err
	}
	if !bytes.Equal(signature, []byte(pngSignatureHex)) {
		return false, fmt.Errorf("invalid PNG signature: %b", signature)
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

	// Step 2: Read 4 bytes of chunk type data.
	readType := make([]byte, 4)
	if _, err := file.Read(readType); err != nil {
		return nil, fmt.Errorf("io.ReadAll failed to read the chunkType")
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
	// TODO: We need to read the chunk data part of the above.
	// This should either be done prior to switching(not after!).
	// Idea is to read in the data, then put whatever data into the relevant
	// structure of that chunk type or ignore it!
	switch chunkType {
	case ChunkIHDR:
		fmt.Println("I'm a IHDR!")
		// Need to extract all the fields, assign them to the IHDR type.

	default:
		fmt.Printf("Skipping chunk type: %s\n", chunkType)
	}

	// TODO: Step 4: Read the CRC and validate it.

	// TODO: Return the chunk
	return &Chunk{
		Length: length,
		Type:   chunkType,
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
