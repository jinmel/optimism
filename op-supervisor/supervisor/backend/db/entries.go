package db

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/entrydb"
)

type searchCheckpoint struct {
	blockNum  uint64
	logIdx    uint32
	timestamp uint64
}

func newSearchCheckpoint(blockNum uint64, logIdx uint32, timestamp uint64) searchCheckpoint {
	return searchCheckpoint{
		blockNum:  blockNum,
		logIdx:    logIdx,
		timestamp: timestamp,
	}
}

func newSearchCheckpointFromEntry(data entrydb.Entry) (searchCheckpoint, error) {
	if data[0] != typeSearchCheckpoint {
		return searchCheckpoint{}, fmt.Errorf("%w: attempting to decode search checkpoint but was type %v", ErrDataCorruption, data[0])
	}
	return searchCheckpoint{
		blockNum:  binary.LittleEndian.Uint64(data[1:9]),
		logIdx:    binary.LittleEndian.Uint32(data[9:13]),
		timestamp: binary.LittleEndian.Uint64(data[13:21]),
	}, nil
}

// encode creates a search checkpoint entry
// type 0: "search checkpoint" <type><uint64 block number: 8 bytes><uint32 event index offset: 4 bytes><uint64 timestamp: 8 bytes> = 20 bytes
func (s searchCheckpoint) encode() entrydb.Entry {
	var data entrydb.Entry
	data[0] = typeSearchCheckpoint
	binary.LittleEndian.PutUint64(data[1:9], s.blockNum)
	binary.LittleEndian.PutUint32(data[9:13], s.logIdx)
	binary.LittleEndian.PutUint64(data[13:21], s.timestamp)
	return data
}

type canonicalHash struct {
	hash TruncatedHash
}

func newCanonicalHash(hash TruncatedHash) canonicalHash {
	return canonicalHash{hash: hash}
}

func newCanonicalHashFromEntry(data entrydb.Entry) (canonicalHash, error) {
	if data[0] != typeCanonicalHash {
		return canonicalHash{}, fmt.Errorf("%w: attempting to decode canonical hash but was type %v", ErrDataCorruption, data[0])
	}
	var truncated TruncatedHash
	copy(truncated[:], data[1:21])
	return newCanonicalHash(truncated), nil
}

func (c canonicalHash) encode() entrydb.Entry {
	var entry entrydb.Entry
	entry[0] = typeCanonicalHash
	copy(entry[1:21], c.hash[:])
	return entry
}

type initiatingEvent struct {
	blockDiff       uint8
	incrementLogIdx bool
	logHash         TruncatedHash
}

func newInitiatingEventFromEntry(data entrydb.Entry) (initiatingEvent, error) {
	if data[0] != typeInitiatingEvent {
		return initiatingEvent{}, fmt.Errorf("%w: attempting to decode initiating event but was type %v", ErrDataCorruption, data[0])
	}
	blockNumDiff := data[1]
	flags := data[2]
	return initiatingEvent{
		blockDiff:       blockNumDiff,
		incrementLogIdx: flags&eventFlagIncrementLogIdx != 0,
		logHash:         TruncatedHash(data[3:23]),
	}, nil
}

func newInitiatingEvent(pre logContext, blockNum uint64, logIdx uint32, logHash TruncatedHash) (initiatingEvent, error) {
	blockDiff := blockNum - pre.blockNum
	if blockDiff > math.MaxUint8 {
		// TODO(optimism#10857): Need to find a way to support this.
		return initiatingEvent{}, fmt.Errorf("too many block skipped between %v and %v", pre.blockNum, blockNum)
	}

	currLogIdx := pre.logIdx
	if blockDiff > 0 {
		currLogIdx = 0
	}
	logDiff := logIdx - currLogIdx
	if logDiff > 1 {
		return initiatingEvent{}, fmt.Errorf("skipped logs between %v and %v", currLogIdx, logIdx)
	}

	return initiatingEvent{
		blockDiff:       uint8(blockDiff),
		incrementLogIdx: logDiff > 0,
		logHash:         logHash,
	}, nil
}

// encode creates an initiating event entry
// type 2: "initiating event" <type><blocknum diff: 1 byte><event flags: 1 byte><event-hash: 20 bytes> = 23 bytes
func (i initiatingEvent) encode() entrydb.Entry {
	var data entrydb.Entry
	data[0] = typeInitiatingEvent
	data[1] = i.blockDiff
	flags := byte(0)
	if i.incrementLogIdx {
		// Set flag to indicate log idx needs to be incremented (ie we're not directly after a checkpoint)
		flags = flags | eventFlagIncrementLogIdx
	}
	data[2] = flags
	copy(data[3:23], i.logHash[:])
	return data
}

func (i initiatingEvent) postContext(pre logContext) logContext {
	post := logContext{
		blockNum: pre.blockNum + uint64(i.blockDiff),
		logIdx:   pre.logIdx,
	}
	if i.blockDiff > 0 {
		post.logIdx = 0
	}
	if i.incrementLogIdx {
		post.logIdx++
	}
	return post
}
