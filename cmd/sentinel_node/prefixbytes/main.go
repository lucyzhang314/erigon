package main

import (
	"bytes"
	"fmt"
	"io"

	"github.com/golang/snappy"
	"github.com/ledgerwatch/erigon/cmd/lightclient/cltypes"
)

var testFinalityUpdatePacket = []uint64{
	74, 38, 197, 139, 200, 4, 255, 6, 0, 0, 115, 78, 97, 80, 112, 89, 1, 76, 2,
	0, 131, 238, 192, 219, 223, 102, 76, 0, 0, 0, 0, 0, 77, 233, 6, 0, 0, 0, 0,
	0, 23, 129, 190, 23, 200, 93, 242, 66, 120, 229, 160, 140, 44, 235, 186, 178,
	29, 205, 96, 89, 53, 116, 194, 49, 244, 112, 117, 38, 36, 37, 251, 126, 97,
	63, 174, 18, 18, 52, 138, 188, 187, 21, 237, 46, 110, 153, 177, 206, 93, 147,
	200, 1, 62, 182, 96, 9, 216, 36, 114, 31, 10, 6, 141, 177, 190, 29, 85, 121,
	52, 37, 35, 69, 233, 144, 147, 216, 230, 178, 39, 180, 238, 25, 66, 58, 53,
	217, 132, 165, 100, 137, 191, 203, 87, 210, 100, 123, 128, 102, 76, 0, 0, 0,
	0, 0, 155, 72, 2, 0, 0, 0, 0, 0, 60, 104, 181, 81, 96, 7, 53, 193, 121, 171,
	251, 213, 199, 67, 65, 168, 213, 36, 54, 82, 23, 231, 12, 3, 29, 19, 172,
	194, 43, 0, 62, 73, 56, 233, 208, 69, 116, 158, 52, 150, 137, 210, 17, 151,
	45, 202, 108, 151, 146, 163, 100, 243, 9, 181, 194, 2, 242, 93, 135, 180, 91,
	63, 85, 176, 127, 234, 171, 59, 71, 118, 11, 178, 67, 104, 73, 168, 85, 181,
	46, 8, 166, 27, 232, 145, 86, 185, 13, 80, 128, 245, 113, 155, 153, 137, 39,
	75, 52, 99, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 65, 101, 88, 22, 4, 244, 176, 47, 93, 194, 65, 37,
	79, 221, 228, 105, 165, 92, 142, 193, 186, 58, 67, 20, 134, 241, 135, 34,
	122, 125, 115, 57, 172, 122, 65, 185, 199, 105, 189, 49, 177, 241, 177, 210,
	33, 99, 244, 214, 26, 95, 84, 189, 174, 224, 193, 54, 236, 63, 221, 184, 78,
	109, 127, 253, 221, 85, 168, 161, 105, 38, 209, 149, 105, 3, 159, 122, 244,
	254, 178, 255, 91, 77, 120, 187, 125, 142, 51, 120, 0, 141, 50, 174, 6, 219,
	151, 239, 0, 85, 229, 188, 59, 64, 96, 194, 166, 103, 55, 135, 128, 213, 137,
	87, 28, 183, 254, 10, 242, 76, 181, 154, 34, 69, 191, 188, 199, 55, 40, 100,
	95, 23, 207, 224, 0, 161, 38, 165, 131, 206, 255, 211, 250, 35, 98, 251, 173,
	131, 104, 203, 253, 249, 177, 151, 226, 215, 171, 138, 240, 89, 169, 36, 255,
	255, 191, 255, 255, 255, 255, 255, 255, 95, 127, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 223, 255, 255, 254, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255, 251, 253, 254, 255, 255, 251, 255,
	255, 247, 255, 249, 223, 191, 255, 255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 129, 125, 109, 155, 212, 25, 158, 44, 114, 150, 201, 26, 87,
	78, 80, 130, 175, 58, 40, 173, 131, 23, 147, 38, 90, 40, 24, 26, 197, 121,
	95, 97, 233, 53, 117, 144, 46, 247, 102, 140, 18, 235, 195, 241, 81, 148,
	154, 224, 23, 5, 47, 142, 181, 239, 122, 137, 144, 31, 160, 122, 11, 36, 129,
	111, 166, 255, 210, 7, 233, 115, 16, 107, 197, 162, 7, 18, 76, 164, 148, 1,
	216, 240, 243, 211, 37, 201, 254, 237, 203, 94, 219, 126, 14, 56, 109, 46,
	224, 102, 76, 0, 0, 0, 0, 0}

func main() {
	// Convery packet to slice of bytes.
	fullPacket := make([]byte, len(testFinalityUpdatePacket))
	for i, v := range testFinalityUpdatePacket {
		fullPacket[i] = byte(v)
	}
	fmt.Printf("Hex Packet: %x\n", fullPacket[:])

	// We are reading a test finality upodate object.
	result := &cltypes.LightClientFinalityUpdate{}
	ln := result.SizeSSZ() // size = 24896

	r := bytes.NewReader(fullPacket)
	// Read first six bytes.
	r.Read(make([]byte, 6))
	sr := snappy.NewReader(r)
	decompressed := make([]byte, ln)
	_, err := io.ReadFull(sr, decompressed)
	if err != nil {
		fmt.Printf("unable to decompress data: %v\n", err)
		return
	}

	err = result.UnmarshalSSZ(decompressed)
	if err != nil {
		fmt.Printf("unable to unmarshall data: %v\n", err)
		return
	}

	fmt.Printf("decoded object: %+v\n", result)
}

/*

func getPrefixFromResponseType(val cltypes.ObjectSSZ) []byte {
	if _, ok := val.(*cltypes.LightClientBootstrap); ok {
		return make([]byte, 7)
	}
	if val.SizeSSZ() <= 16 {
		return []byte{0x08}
	}
	return []byte{0x4a, 0x26, 0xc5, 0x8b, 0xc8, 0x04}
}

func DecodeAndRead(r io.Reader, val cltypes.ObjectSSZ) error {
	ln := val.SizeSSZ()
	if _, err := r.Read(getPrefixFromResponseType(val)); err != nil {
		return err
	}

	sr := snappy.NewReader(r)
	raw := make([]byte, ln)
	_, err := io.ReadFull(sr, raw)

	if err != nil {
		return fmt.Errorf("readPacket: %w", err)
	}

	err = val.UnmarshalSSZ(raw)

	return err
}

*/
