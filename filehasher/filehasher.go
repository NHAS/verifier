package filehasher

import (
	"Verifier/threadmgmt"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"

	"os"
	"path/filepath"
)

type WalkFunc func(path string, info os.FileInfo, err error) error

type workerGroup struct {
	startPath string
	resultVal []byte
}

func Start(root string, numConsumer int) ([]byte, error) {
	wg := workerGroup{startPath: root}

	err := threadmgmt.Start(wg.producer, wg.consumer, wg.collectionCounter, wg.result, numConsumer)
	if err != nil {
		return nil, err
	}

	return wg.resultVal, nil
}

func (wg *workerGroup) producer(paths chan<- interface{}) error {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return err
	}

	return Walk(wg.startPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && info.Mode().IsRegular() {
			relativePath, err := filepath.Rel(workingDirectory, path)
			if err != nil {
				paths <- path
			} else {
				paths <- relativePath
			}
		}

		return nil
	})

}

func (wg *workerGroup) consumer(input interface{}) (key string, value string, err error) {

	switch path := input.(type) {

	case string:

		hash := sha256.New()
		file, err := os.Open(path)
		if err != nil {
			return path, err.Error(), err
		}

		if _, err := io.Copy(hash, file); err != nil {
			return path, err.Error(), err
		}
		file.Close()

		return path, hex.EncodeToString(hash.Sum(nil)), nil

	default:
		return "", "", threadmgmt.BadType
	}
}

func (wg *workerGroup) collectionCounter(key, val string) {
	log.Println(key, " = ", val)
}

func (wg *workerGroup) result(filesmapping map[string]string) {
	js, err := json.Marshal(filesmapping)
	if err != nil {
		fmt.Println("Error marshling json:", err)
		return
	}

	wg.resultVal = js
}

func walk(path string, info os.FileInfo, walkFn WalkFunc) error {

	if !info.IsDir() {

		return walkFn(path, info, nil)

	}

	names, err := readDirNames(path)

	err1 := walkFn(path, info, err)

	// If err != nil, walk can't walk into this directory.

	// err1 != nil means walkFn want walk to skip this directory or stop walking.

	// Therefore, if one of err and err1 isn't nil, walk will return.

	if err != nil || err1 != nil {

		// The caller's behavior is controlled by the return value, which is decided

		// by walkFn. walkFn may ignore err and return nil.

		// If walkFn returns SkipDir, it will be handled by the caller.

		// So walk should return whatever walkFn returns.

		return err1

	}

	for _, name := range names {

		filename := filepath.Join(path, name)

		fileInfo, err := os.Lstat(filename)

		if err != nil {

			if err := walkFn(filename, fileInfo, err); err != nil {

				return err

			}

		} else {

			err = walk(filename, fileInfo, walkFn)

			if err != nil {

				if !fileInfo.IsDir() {

					return err

				}

			}

		}

	}

	return nil

}

// Walk walks the file tree rooted at root, calling walkFn for each file or

// directory in the tree, including root. All errors that arise visiting files

// and directories are filtered by walkFn. The files are walked in lexical

// order, which makes the output deterministic but means that for very

// large directories Walk can be inefficient.

// Walk does not follow symbolic links.

func Walk(root string, walkFn WalkFunc) error {

	info, err := os.Lstat(root)

	if err != nil {

		err = walkFn(root, nil, err)

	} else {

		err = walk(root, info, walkFn)

	}

	return err

}

// readDirNames reads the directory named by dirname and returns

// a sorted list of directory entries.

func readDirNames(dirname string) ([]string, error) {

	f, err := os.Open(dirname)

	if err != nil {

		return nil, err

	}

	names, err := f.Readdirnames(-1)

	f.Close()

	if err != nil {

		return nil, err

	}

	return names, nil

}
