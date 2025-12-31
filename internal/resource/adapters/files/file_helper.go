package files

import (
	"fmt"
	"io"
	"os" // os was used in the function
)

// compareFiles iki dosyanın içeriğini karşılaştırır.
func compareFiles(src, dst string) (bool, error) {
	sInfo, err := os.Stat(src)
	if err != nil {
		return false, err
	}
	dInfo, err := os.Stat(dst)
	if err != nil {
		return false, err
	}
	// Boyut farkı varsa direkt farklıdır
	if sInfo.Size() != dInfo.Size() {
		return false, nil
	}

	// Byte byte karşılaştır
	f1, err := os.Open(src)
	if err != nil {
		return false, err
	}
	defer f1.Close()

	f2, err := os.Open(dst)
	if err != nil {
		return false, err
	}
	defer f2.Close()

	// 4KB buffer ile karşılaştır
	const chunkSize = 4096
	b1 := make([]byte, chunkSize)
	b2 := make([]byte, chunkSize)

	for {
		n1, err1 := f1.Read(b1)
		n2, err2 := f2.Read(b2)

		if err1 != nil || err2 != nil {
			if err1 == io.EOF && err2 == io.EOF {
				return true, nil
			}
			if err1 == io.EOF || err2 == io.EOF {
				return false, nil
			}
			return false, fmt.Errorf("read error: %v, %v", err1, err2)
		}

		if n1 != n2 || string(b1[:n1]) != string(b2[:n2]) {
			return false, nil
		}
	}
}
