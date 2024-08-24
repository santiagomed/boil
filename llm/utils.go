package llm

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"time"
)

func generateBatchID() string {
	timestamp := time.Now().Unix()
	randomBytes := make([]byte, 8)
	rand.Read(randomBytes)

	id := make([]byte, 12)
	binary.BigEndian.PutUint32(id[:4], uint32(timestamp))
	copy(id[4:], randomBytes)

	return hex.EncodeToString(id)
}

func isValidBatchID(s string) bool {
	_, err := hex.DecodeString(s)
	return err == nil && len(s) == 24
}

func EnsureBatchID(s string) string {
	if !isValidBatchID(s) {
		return generateBatchID()
	}
	return s
}
