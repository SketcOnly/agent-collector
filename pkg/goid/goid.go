package goid

import "runtime"

// getGID 获取当前 goroutine 的 ID（高性能版本）
func GetGID() uint64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	b := buf[:n]
	// 栈信息类似: "goroutine 123 [running]:\n"
	var id uint64
	for i := 10; i < len(b); i++ {
		c := b[i]
		if c < '0' || c > '9' {
			break
		}
		id = id*10 + uint64(c-'0')
	}
	return id
}
