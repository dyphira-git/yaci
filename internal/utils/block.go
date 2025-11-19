package utils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/manifest-network/yaci/internal/client"
	"github.com/pkg/errors"
)

const statusMethod = "cosmos.base.node.v1beta1.Service.Status"
const getBlockByHeightMethod = "cosmos.base.tendermint.v1beta1.Service.GetBlockByHeight"

// GetLatestBlockHeightWithRetry gets current block height from Status endpoint
func GetLatestBlockHeightWithRetry(gRPCClient *client.GRPCClient, maxRetries uint) (uint64, error) {
	return ExtractGRPCField(
		gRPCClient,
		statusMethod,
		maxRetries,
		"height",
		func(s string) (uint64, error) {
			height, err := strconv.ParseUint(s, 10, 64)
			if err != nil {
				return 0, errors.WithMessage(err, "error parsing height")
			}
			return height, nil
		},
	)
}

// GetEarliestBlockHeight determines earliest available block on node.
// Returns 1 for archive nodes, or parses error message for pruned nodes.
func GetEarliestBlockHeight(gRPCClient *client.GRPCClient, maxRetries uint) (uint64, error) {
	inputParams := []byte(`{"height":"1"}`)
	_, err := GetGRPCResponse(gRPCClient, getBlockByHeightMethod, 1, inputParams)

	// Block 1 exists - archive node with full history
	if err == nil {
		return 1, nil
	}

	// Extract lowest height from error: "height 1 is not available, lowest height is 28566001"
	lowestHeight := parseLowestHeightFromError(err.Error())
	if lowestHeight > 0 {
		return lowestHeight, nil
	}

	// Retry with full retries if error was transient
	_, err = GetGRPCResponse(gRPCClient, getBlockByHeightMethod, maxRetries, inputParams)
	if err == nil {
		return 1, nil
	}

	return 0, fmt.Errorf("failed to determine earliest block height: %w", err)
}

// parseLowestHeightFromError extracts lowest height from pruned node errors
func parseLowestHeightFromError(errMsg string) uint64 {
	re := regexp.MustCompile(`lowest height is (\d+)`)
	matches := re.FindStringSubmatch(strings.ToLower(errMsg))

	if len(matches) >= 2 {
		height, err := strconv.ParseUint(matches[1], 10, 64)
		if err == nil {
			return height
		}
	}

	return 0
}
