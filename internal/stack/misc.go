package stack

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
)

var _randomSeparator struct {
	sync.Once
	value string
}

func randomSeparator() (string, error) {
	var err error
	_randomSeparator.Do(func() {
		buf := make([]byte, 32)
		_, err = rand.Read(buf)
		if err != nil {
			panic(err)
		}
		_randomSeparator.value = hex.EncodeToString(buf)
	})
	return _randomSeparator.value, err
}
