package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"hash/crc32"
	"net"
	"os"
	"sort"
)

type Block struct {
	Index    uint32
	Data     []byte
	Checksum uint32
}

func main() {
	multicastAddress := flag.String("address", "224.3.29.71:10000", "Multicast address and port")
	savePath := flag.String("save", "received_file", "Path to save the received file")
	flag.Parse()

	addr, err := net.ResolveUDPAddr("udp4", *multicastAddress)
	if err != nil {
		panic(err)
	}

	conn, err := net.ListenMulticastUDP("udp4", nil, addr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	blocks := make(map[uint32]Block)
	var totalBlocks uint32

	buffer := make([]byte, 4096+8)
	fmt.Println("Listening for multicast messages...")

	for {
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			panic(err)
		}

		blockIndex := binary.BigEndian.Uint32(buffer[:4])
		receivedChecksum := binary.BigEndian.Uint32(buffer[4:8])
		data := buffer[8:n]

		calculatedChecksum := crc32.ChecksumIEEE(data)
		if receivedChecksum != calculatedChecksum {
			fmt.Printf("Checksum mismatch for block %d.\n", blockIndex)
			continue
		}

		if blockIndex == 0 {
			totalBlocks = binary.BigEndian.Uint32(data[:4])
			fmt.Printf("Primary block received. Total Blocks: %d\n", totalBlocks)
		} else {
			blocks[blockIndex] = Block{Index: blockIndex, Data: data, Checksum: receivedChecksum}
		}

		if uint32(len(blocks)) == totalBlocks-1 {
			writeBlocksInOrder(blocks, *savePath, totalBlocks)
			break
		}
	}
}

func writeBlocksInOrder(blocks map[uint32]Block, savePath string, totalBlocks uint32) {
	file, err := os.Create(savePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	var indices []int
	for index := range blocks {
		indices = append(indices, int(index))
	}
	sort.Ints(indices)

	for _, index := range indices {
		block := blocks[uint32(index)]
		_, err := file.Write(block.Data)
		if err != nil {
			panic(err)
		}
	}

	fmt.Println("File successfully reconstructed and saved.")
}
