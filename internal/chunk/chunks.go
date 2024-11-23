package chunk

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// Chunk defines the chunk layout as specified by PNG datastream structure.
type Chunk struct {
	Length uint32    // A four-byte unsigned integer giving the number of bytes in the chunk's data field.
	Type   ChunkType // A sequence of four bytes defining the chunk type.
	Data   []byte    // The data bytes of the relevant chunk type; can be zero length.
	Crc    uint32    // A four-byte CRC (Cyclic Redundancy Code) calculated on the preceding bytes in the chunk.
	// Includes chunk type and data, but NOT length.
}

type IHDR struct {
	Width             uint32
	Height            uint32
	BitDepth          uint8
	ColorType         uint8
	CompressionMethod uint8
	FilterMethod      uint8
	InterlaceMethod   uint8
}

func HandleIHDR(chunkStream *Chunk) (IHDR, error) {
	if len(chunkStream.Data) != 13 {
		return IHDR{}, fmt.Errorf("invalid length for IHDR:", len(chunkStream.Data))
	}
	return IHDR{
		Width:             binary.BigEndian.Uint32(chunkStream.Data[0:4]),
		Height:            binary.BigEndian.Uint32(chunkStream.Data[4:8]),
		BitDepth:          chunkStream.Data[8],
		ColorType:         chunkStream.Data[9],
		CompressionMethod: chunkStream.Data[10],
		FilterMethod:      chunkStream.Data[11],
		InterlaceMethod:   chunkStream.Data[12],
	}, nil
}

func HandleIDAT(chunkStream *Chunk, dest io.Writer) error {
	_, err := dest.Write(chunkStream.Data)
	if err != nil {
		return fmt.Errorf("error writing to IDAT buffer: %v", err)
	}
	return nil
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
