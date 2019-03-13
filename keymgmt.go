// keymgmt
package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"golang.org/x/crypto/ed25519"
)

const defaultKeyPath string = "$HOME/.vkeys/default"

func checkKey(path string) (privkey ed25519.PrivateKey, err error) {
	if path == defaultKeyPath {
		path = os.Getenv("HOME") + "/.vkeys/"
		_ = os.Mkdir(path, 0744)
		path += "default"
	}

	if _, err := os.Stat(path); err == nil {

		bytes, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}

		block, _ := pem.Decode(bytes)
		if block == nil || block.Type != "ED25519 KEY" {
			return nil, errors.New("Key could not be loaded as an ed25519 key")
		}

		privkey = block.Bytes

	} else if os.IsNotExist(err) {
		fmt.Println("Key does not exist, generating new ed25519 (32 bits)")
		file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0600)
		if err != nil {
			return nil, err
		}

		_, privkey, err = ed25519.GenerateKey(rand.Reader)

		block := &pem.Block{
			Type:  "ED25519 KEY",
			Bytes: []byte(privkey),
		}

		pem.Encode(file, block)
		file.Close()

	} else if os.IsPermission(err) {
		fmt.Println("Bad permissions on ", path)
		return nil, err
	} else {
		return nil, err
	}

	return privkey, nil
}

func addSignature(toSign []byte, privkey ed25519.PrivateKey) []byte {

	sig := []byte(hex.EncodeToString(ed25519.Sign(privkey, toSign)))

	verificationFileBytes := append(sig, []byte("\n")...)
	verificationFileBytes = append(verificationFileBytes, toSign...)

	return verificationFileBytes
}

func checkSignature(toCheck []byte, privkey ed25519.PrivateKey) ([]byte, error) {
	parts := bytes.SplitN(toCheck, []byte("\n"), 2)
	signature, err := hex.DecodeString(strings.TrimSpace(string(parts[0])))
	if err != nil {
		return nil, err
	}

	message := parts[1]

	if !ed25519.Verify(privkey.Public().(ed25519.PublicKey), message, signature) {
		return nil, errors.New("Signature could not be verified on file")
	}

	return message, nil
}
