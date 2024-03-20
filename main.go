package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"math"
	"math/big"
	"os"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/fananchong/fastlz-go"
)

var (
	analysisMode = true
	baseMainnetURL = "https://base-rpc.publicnode.com"
	baseSepoliaURL = "https://sepolia.base.org"
)

func main() {
	if analysisMode {
		// If we are in analysis mode, we don't need to run the full node.
		analysis("result.csv")
		return
	}

	client, err := ethclient.DialContext(context.Background(), baseMainnetURL)
	if err != nil {
		panic(err)
	}

	block, err := client.BlockByNumber(context.Background(), nil)
	if err != nil {
		panic(err)
	}

	pastBlocks := 100000
	startBlock := block.NumberU64() - uint64(pastBlocks)

	fmt.Println("start block:", startBlock)

	for i := uint64(11801250); i < 11801250+1; i++ {
		block, err := client.BlockByNumber(context.Background(), new(big.Int).SetUint64(i))
		if err != nil {
			panic(err)
		}
		for _, tx := range block.Transactions() {
			data, err := tx.MarshalBinary()
			if err != nil {
				panic(err)
			}

			out := types.NewRollupCostData(data)

			if types.DepositTxType == tx.Type() {
				continue
			}

			output := make([]byte, int(math.Max(66, float64(len(data))*1.05)))
			resultLen := fastlz.Fastlz_compress(data, len(data), output)
			// fmt.Println(output, 1280000, output-uint32(1280000), resultLen, resultLen-int(output), i)

			fmt.Println(fmt.Sprintf("%d\t%d\t%d\t%d\t%s", block.NumberU64(), len(data), out.CompressedSize, resultLen, tx.Hash().Hex()))
		}
	}
}

func analysis(fileName string) {
	file, err := os.Open(fileName)
	if err != nil {
		panic(err)
	}

	defer file.Close()

	// Create a CSV reader
	reader := csv.NewReader(file)

	// Read and process each line
	lines, err := reader.ReadAll()

	var notCompressed int64
	// txSize/255+16
	var overHead1CannotCovered int64
	var overHead2CannotCovered int64
	var smallestOverhead int64 = 10000000
	var smallestLine int
	for i, line := range lines {
		if i == 0 {
			continue
		}

		line = strings.Split(line[0], "\t")
		original, err := strconv.ParseInt(line[1], 10, 64)
		if err != nil {
			panic(err)
		}
		compressed, err := strconv.ParseInt(line[2], 10, 64)
		if err != nil {
			panic(err)
		}

		if compressed > original {
			notCompressed++
		}

		if original+original/255+16 < compressed {
			overHead1CannotCovered++
		}

		if original+int64(float64(original)*0.03) < compressed {
			overHead2CannotCovered++
		}

		if original+int64(float64(original)*0.03)-compressed < smallestOverhead {
			smallestOverhead = original + int64(float64(original)*0.03) - compressed
			smallestLine = i
			fmt.Println("overhead1 cannot covered", line[0], original, compressed, original+int64(float64(original)*0.04)-compressed, smallestOverhead)
		}

	}

	fmt.Println(fmt.Sprintf("total:%d, not compressed: %d, overhead1: %d, overhead2: %d, worst case:%d, line: %d", len(lines)-1, notCompressed, overHead1CannotCovered, overHead2CannotCovered, smallestOverhead, smallestLine))
}
