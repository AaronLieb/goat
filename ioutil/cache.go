package ioutil

import (
	"fmt"
	"os"

	"github.com/AaronLieb/goat/util"
)

func ReadCache(name string) ([]byte, error) {
	return os.ReadFile(fmt.Sprintf("%s/%s.out", tempDir(), name))
}

func WriteCache(name string, data string) error {
	f, err := OpenCache(name)
	if err != nil {
		return err
	}
	fmt.Fprint(f, data)
	return nil
}

func OpenCache(name string) (*os.File, error) {
	dir := tempDir()
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return nil, err
	}

	f, err := os.Create(fmt.Sprintf("%s/%s.out", dir, name))
	if err != nil {
		return nil, err
	}
	return f, nil
}

func tempDir() string {
	return os.TempDir() + util.CommandName
}
