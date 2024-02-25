package files

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

type Store struct {
	url  string
	path string
}

func NewStore(url, path string) Store {
	return Store{url, path}
}

func (s Store) Save(buffer []byte, relativePath string) (string, error) {
	file, fullPath, err := s.createFile(relativePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	written, err := file.Write(buffer)
	if err != nil {
		return "", fmt.Errorf("error writing to file with path: '%s'. Error: %s", fullPath, err)
	}
	if written != len(buffer) {
		return "", fmt.Errorf("written to file %d bytes. Expected to write %d bytes", written, len(buffer))
	}

	return s.generateUrl(relativePath), nil
}

func (s Store) generateUrl(relativePath string) string {
	return fmt.Sprintf("%s/%s", s.url, relativePath)
}

func (s Store) Copy(searchPath, targetPath string, other Store) (string, error) {
	outFile, _, err := s.openFile(searchPath)
	if err != nil {
		return "", err
	}
	defer outFile.Close()
	inFile, _, err := other.createFile(targetPath)
	if err != nil {
		return "", err
	}
	defer inFile.Close()
	reader := bufio.NewReader(outFile)
	writer := bufio.NewWriter(inFile)
	if _, err = reader.WriteTo(writer); err != nil {
		return "", err
	}
	return other.generateUrl(targetPath), nil
}

func (s Store) Exists(relativePath string) bool {
	_, _, err := s.openFile(relativePath)
	return err == nil
}

func (s Store) openFile(relativePath string) (*os.File, string, error) {
	fullPath := fmt.Sprintf("%s/%s", s.path, relativePath)
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fullPath, err
	}
	return file, fullPath, nil
}

func (s Store) createFile(relativePath string) (*os.File, string, error) {
	fullPath := fmt.Sprintf("%s/%s", s.path, relativePath)
	fileNameStart := strings.LastIndex(fullPath, "/")
	os.MkdirAll(fullPath[:fileNameStart], os.ModePerm)
	var file *os.File
	if _, err := os.Stat(fullPath); errors.Is(err, os.ErrNotExist) {
		if file, err = os.Create(fullPath); err != nil {
			return nil, fullPath, fmt.Errorf("cannot create new file with path: '%s'. Error: %s", fullPath, err)
		}
	} else {
		if file, err = os.OpenFile(fullPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm); err != nil {
			return nil, fullPath, fmt.Errorf("cannot open file with path: '%s'. Error: %s", fullPath, err)
		}
	}
	return file, fullPath, nil
}

func (s Store) DeleteOld(duration time.Duration) error {
	files, err := os.ReadDir(s.path)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.Type().IsRegular() {
			info, err := file.Info()
			if err != nil {
				return err
			}
			if time.Since(info.ModTime()) > duration {
				if err := os.Remove(file.Name()); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (s Store) Open(path string) (io.ReadWriteCloser, string, error) {
	return s.createFile(path)
}

func (s Store) Delete(relativePath string) error {
	fullPath := fmt.Sprintf("%s/%s", s.path, relativePath)
	return os.Remove(fullPath)
}
