// Verifier project main.go
package main

import (
	"Verifier/filehasher"
	"Verifier/verify"

	"flag"
	"fmt"
	"io/ioutil"
	"os"
)

func main() {

	directory := flag.String("d", "", "Directory to perform operation")

	verificationFile := flag.String("f", "verify.vf", " Verification filename")

	verifyFiles := flag.Bool("v", false, "Whether to verify or not")

	keyPtr := flag.String("k", defaultKeyPath, "Path to key")

	numConsumers := flag.Int("t", 20, "Number of threads to use")

	flag.Parse()

	if len(*directory) == 0 && !*verifyFiles {
		fmt.Println("No directory specified with flag -d")
		flag.PrintDefaults()
		os.Exit(1)
	}

	privkey, err := checkKey(*keyPtr)
	if err != nil {
		fmt.Println("Error loading key: ", *keyPtr, err)
		os.Exit(1)
	}

	if !*verifyFiles {

		filesList, err := filehasher.Start(*directory, *numConsumers)
		if err != nil {
			fmt.Println("Error hashing files: ", err)
			os.Exit(1)
		}

		verificationFileBytes := addSignature(filesList, privkey)

		err = ioutil.WriteFile(*verificationFile, verificationFileBytes, 0644)
		if err != nil {
			fmt.Println("Error writing file to directory: ", err)
			os.Exit(1)
		}

	} else {
		if len(*directory) == 0 {
			wd, err := os.Getwd()

			if err != nil {
				fmt.Println("Unable to get working directory: ", err)
				os.Exit(1)
			}
			directory = &wd
		}

		filelist, err := ioutil.ReadFile(*directory + "/" + *verificationFile)
		if err != nil {
			fmt.Println("Unable to read file: ", err)
			os.Exit(1)
		}

		filelist, err = checkSignature(filelist, privkey)
		if err != nil {
			fmt.Println("Verifying signature failed: ", err)
			os.Exit(2)
		}

		nonMatching, err := verify.Start(filelist, *numConsumers)
		if err != nil {
			fmt.Println("Unable to start verifying hashes: ", err)
			os.Exit(1)
		}

		if len(nonMatching) != 0 {
			fmt.Print("\nVerifying files failed.\n")
			fmt.Println("Failed: ")
			for k, v := range nonMatching {
				fmt.Print("\"", k, "\" Reason: \"", v, "\"\n")
			}

			os.Exit(2)

		} else {
			fmt.Println("Verifying files succeeded!")
		}

	}

	os.Exit(0)

}
