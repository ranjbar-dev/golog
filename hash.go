package golog

import (
	"crypto/sha256"
	"fmt"
	"time"
)

func (l *GoLog) generateHash() string {

	timestmap := time.Now().Unix()

	key := fmt.Sprintf("%d%s", timestmap/10, l.config.ServerKey)

	data := []byte(key)

	return fmt.Sprintf("%x", sha256.Sum256(data))
}
