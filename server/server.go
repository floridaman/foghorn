package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"hash/crc32"
	"io"
	"math"
	"net"
	"os"
	"time"
)

// Block represents a data block to be sent.
type Block struct {
	Index    uint32
	Checksum uint32
	Data     []byte
}

func main() {
	// Command line arguments
	filePath := flag.String("file", "path/to/your/file", "Path to the file to be sent")
	multicastAddress := flag.String("address", "224.3.29.71:10000", "Multicast address and port")
	blockSize := flag.Int("size", 508, "Block size in bytes")
	delay := flag.Int("delay", 100, "Delay in ms")

	flag.Parse()

	// Open file
	file, err := os.Open(*filePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		panic(err)
	}

	// Calculate total blocks (+2 for possible partial block and primary block)
	fileSize := fileInfo.Size()
	totalBlocks := math.Ceil(float64(fileSize) / float64(*blockSize))

	// Calculate file checksum
	hash := crc32.NewIEEE()
	if _, err := file.Seek(0, 0); err != nil { // Seek to start
		panic(err)
	}
	if _, err := io.Copy(hash, file); err != nil {
		panic(err)
	}
	fileChecksum := hash.Sum32()

	// Prepare primary block
	primaryBlock := Block{
		Index:    0, // Primary block index 0
		Checksum: fileChecksum,
		Data:     make([]byte, 12), // Placeholder for totalBlocks and fileChecksum
	}
	binary.BigEndian.PutUint32(primaryBlock.Data[0:4], uint32(totalBlocks))
	binary.BigEndian.PutUint64(primaryBlock.Data[4:12], uint64(fileChecksum))

	// Setup UDP connection
	addr, err := net.ResolveUDPAddr("udp4", *multicastAddress)
	if err != nil {
		panic(err)
	}
	conn, err := net.DialUDP("udp4", nil, addr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	fmt.Printf("Sending file: %s\n", *filePath)
	fmt.Printf("Multicast Address: %s\n", *multicastAddress)
	fmt.Printf("Block Size: %d bytes\n", *blockSize)

	currentBlock := uint32(0)

	// Start a goroutine for status updates
	go func() {
		for {
			fmt.Printf("\rTransmitting block %d of %d", currentBlock, totalBlocks)
			time.Sleep(1 * time.Second) // Simulate block transmission time
		}
	}()

	// Infinite loop to send file blocks
	for {
		// Send primary block
		sendBlock(conn, primaryBlock)

		// Reset file pointer to beginning for block transmission
		if _, err := file.Seek(0, 0); err != nil {
			panic(err)
		}

		// Block transmission
		buffer := make([]byte, *blockSize)
		blockIndex := uint32(1) // Start from 1, 0 is for primary block
		for {
			bytesRead, err := file.Read(buffer)
			if err == io.EOF {
				break
			}
			if err != nil {
				panic(err)
			}

			// Calculate block checksum
			// hash.Reset()
			// hash.Write(buffer[:bytesRead])
			// blockChecksum := hash.Sum32()

			// Prepare and send block
			block := Block{
				Index: blockIndex,
				// Checksum: blockChecksum,
				Data: buffer[:bytesRead],
			}
			sendBlock(conn, block)
			blockIndex++
			currentBlock = blockIndex
		}

		// Delay between each file transmission
		time.Sleep(time.Duration(*delay) * time.Millisecond)
	}
}

// sendBlock sends a block over UDP.
func sendBlock(conn *net.UDPConn, block Block) {
	header := make([]byte, 8)
	binary.BigEndian.PutUint32(header[0:4], block.Index)
	binary.BigEndian.PutUint32(header[4:8], block.Checksum)
	if _, err := conn.Write(append(header, block.Data...)); err != nil {
		panic(err)
	}
}
