package verify

import (
	"github.com/NHAS/verifier/threadmgmt"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"

	"os"
	"strconv"
)

type workerGroup struct {
	work           map[string]string
	resultVal      map[string]string
	itemsProcessed int
}

type hashPair struct {
	path string
	hash string
}

func Start(file []byte, numConsumer int) (map[string]string, error) {

	wg := workerGroup{}
	err := json.Unmarshal(file, &wg.work)
	if err != nil {
		return nil, err
	}

	err = threadmgmt.Start(wg.producer, wg.consumer, wg.collectorCounter, wg.result, numConsumer)
	if err != nil {
		return nil, err
	}
	fmt.Print("\n")

	return wg.resultVal, nil
}

func (wg *workerGroup) producer(hashpairs chan<- interface{}) error {
	for key, value := range wg.work {
		hashpairs <- hashPair{key, value}
	}
	return nil
}

func (wg *workerGroup) consumer(input interface{}) (key string, value string, err error) {
	switch hashpair := input.(type) {
	case hashPair:

		hash := sha256.New()
		file, err := os.Open(hashpair.path)
		if err != nil {
			return hashpair.path, err.Error(), err
		}

		if _, err := io.Copy(hash, file); err != nil {
			return hashpair.path, err.Error(), err
		}
		file.Close()

		hashVal := hex.EncodeToString(hash.Sum(nil))

		if hashVal != hashpair.hash {
			return hashpair.path, hashpair.hash, nil
		}

	default:
		return "", "", threadmgmt.BadType

	}

	return "", "", threadmgmt.CountButSkipCollecting // If the hashes match, then no need to report them in the result
}

func (wg *workerGroup) collectorCounter(key, value string) {
	wg.itemsProcessed++

	for i := 0; i < len("Verifying files...")+len(strconv.Itoa(wg.itemsProcessed)); i++ {
		fmt.Print("\r")
	}

	fmt.Print("Verifying files...", wg.itemsProcessed)

}

func (wg *workerGroup) result(badmatches map[string]string) {
	wg.resultVal = badmatches
}
