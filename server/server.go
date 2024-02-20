package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"hash/crc32"
	"net"
	"os"
	"time"
)

// Block represents a data block to be sent.
type Block struct {
	Index       int
	TotalBlocks int
	Hash        uint32
	Data        []byte
}

func main() {
	// Command line arguments
	filePath := flag.String("file", "", "Path to the file to be sent")
	multicastAddress := flag.String("address", "224.3.29.71:10000", "Multicast address and port")
	blockSize := flag.Int("size", 508, "Block size in bytes")
	delay := flag.Int("delay", 100, "Delay in ms")

	flag.Parse()

	if *filePath == "" {
		fmt.Println("File path is required")
		return
	}

	// Read the entire file into memory
	fileData, err := os.ReadFile(*filePath)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}

	// Calculate total blocks
	totalBlocks := (len(fileData) + *blockSize - 1) / *blockSize

	// Calculate file checksum
	hash := crc32.ChecksumIEEE(fileData)

	// Prepare primary block
	primaryBlock := Block{
		Index:       0,
		TotalBlocks: totalBlocks,
		Hash:        hash,
	}

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

	currentBlock := 0

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

		// blockIndex := uint32(1) // Start from 1, 0 is for primary block

		for blockIndex := 1; blockIndex < len(fileData); blockIndex += *blockSize {
			endIndex := blockIndex + *blockSize
			if endIndex > len(fileData) {
				endIndex = len(fileData)
			}

			// Prepare and send block
			block := Block{
				Index: blockIndex,
				Data:  fileData[blockIndex:endIndex],
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
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header[0:4], uint32(block.Index))

	if _, err := conn.Write(append(header, block.Data...)); err != nil {
		panic(err)
	}
}
