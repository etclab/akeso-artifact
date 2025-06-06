package keymanager

import (
	"fmt"
	"testing"
)

var sink []byte

// this measures:
// 1. time taken to create the message (the shared AES key) +
// 2. time taken to encrypt the message for all members +
// 3. time taken to decrypt the message for all members
func BenchmarkKeyRotation(b *testing.B) {
	for i := 1; i <= 20; i++ {
		size := 1 << i
		pool := &Pool{}
		pool.Init(size)

		b.Run(fmt.Sprintf("Group-%d", size), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				msg := pool.RotateOnce()
				sink = msg // fake use to prevent compiler optimization
			}
		})
	}
}
