package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"flag"
	"fmt"
	"image"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/snksoft/crc"
	"png.adpollak.net/internal/chunk"
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
	var pngCLI string
	flag.StringVar(&pngCLI, "png", defaultFilePath, "png file to supply")

	// NOTE:
	flag.Parse()

	// Open the png
	file, err := os.Open(pngCLI)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer file.Close()

	log.Printf("Successfully opened %s\n", pngCLI)

	decoder := NewPngDecoder()
	if _, err := decoder.IsPng(file); err != nil {
		log.Fatal(err)
	}

	_, err = decoder.ParseChunkStream(file)
	if err != nil {
		log.Fatal(err)
	}

	/*
		f, err := os.Create("image.png")
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		err = png.Encode(f, img)
		if err != nil {
			log.Fatal(err)
		}
	*/
	log.Println("PNG file parsed successfully!")
}

// PngDecoder represents the process of reconstructing the reference image from a
// PNG datastream and generating a delivered image.
type PngDecoder struct {
	Chunks []chunk.Chunk
}

// NewPngDecoder creates a new PngDecoder
func NewPngDecoder() *PngDecoder {
	return &PngDecoder{
		Chunks: make([]chunk.Chunk, 0),
	}
}

// ParseChunkStream parses (decodes) a PNG datastream based on the sequence of
// chunks read in.
// NOTE: This method assumes a valid PNG file has already been passed in.
func (p *PngDecoder) ParseChunkStream(file *os.File) (image.Image, error) {
	// idat is a buffer to hold idat chunk data
	// UPDATE: use a bytes.Buffer type instead of []byte for efficient writing to the buffer.
	var idat bytes.Buffer
	var ihdr chunk.IHDR
loop:
	for {
		chunkStream, err := p.readChunk(file)
		if err != nil {
			return nil, err
		}
		switch chunkStream.Type {
		case chunk.ChunkIHDR:
			ihdr = chunk.IHDR{
				Width:             binary.BigEndian.Uint32(chunkStream.Data[0:4]),
				Height:            binary.BigEndian.Uint32(chunkStream.Data[4:8]),
				BitDepth:          chunkStream.Data[8],
				ColorType:         chunkStream.Data[9],
				CompressionMethod: chunkStream.Data[10],
				FilterMethod:      chunkStream.Data[11],
				InterlaceMethod:   chunkStream.Data[12],
			}
			log.Printf("IHDR data: %+v\n", ihdr)
			// TODO: handle image type based on color type
		case chunk.ChunkIDAT:
			_, err := idat.Write(chunkStream.Data)
			if err != nil {
				return nil, fmt.Errorf("error writing to IDAT buffer: %v", err)
			}
			log.Println("Reached IDAT")
		case chunk.ChunkIEND:
			log.Println("Reached IEND")
			break loop
		}
	}
	// TODO: Use zlib to inflate the IDAT data.

	// Inflate the deflate'd data from idat.
	inflatedData, err := zlib.NewReader(&idat)
	if err != nil {
		return nil, fmt.Errorf("failed to read deflated IDAT data: %v", err)
	}
	defer inflatedData.Close()

	// Read inflatedData into a bytes Buffer.
	var decompressedBytes bytes.Buffer
	if _, err := io.Copy(&decompressedBytes, inflatedData); err != nil {
		return nil, fmt.Errorf("error reading inflatedData: %v", err)
	}

	// TODO: create the image dependent on color type as stated from IHDR.
	// pixels := decompressedBytes.Bytes() // Transform the bytes Buffer into a slice to work with the image data

	return nil, nil
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
func (p *PngDecoder) readChunk(file *os.File) (*chunk.Chunk, error) {
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
	chunkType, err := chunk.FromString(chunkBuffer)
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

	// Extract the chunk data and store (or ignore) for relevant chunk type.
	switch chunkType {
	case chunk.ChunkIHDR:
		log.Printf("Parsed IHDR\n")
	case chunk.ChunkIDAT:
		log.Printf("Parsed IDAT\n")
	case chunk.ChunkIEND:
		log.Println("IEND. Done!")
	default:
		fmt.Printf("Skipping chunk type: %s\n", chunkType)
	}

	return &chunk.Chunk{
		Length: length,
		Type:   chunk.ChunkType(chunkType),
		Data:   chunkData,
		Crc:    uint32(computedCRC),
	}, nil
}

// TODO: Create separate functions or methods to handle chunk types
