package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/hashgraph/hedera-sdk-go/v2"
	"github.com/joho/godotenv"
)

/*
	This signer acts as an external wallet. It receives the transaction bytes, signs with the account's private key, and then returns
	the signed transaction bytes for execution in the other application.
*/

func init() {
	// load .env file on init
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func main() {
	http.HandleFunc("/sign", func(w http.ResponseWriter, r *http.Request) {
		// read the request body (transaction bytes)
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Transaction received")

		// get account private key from environment variable
		privateKey, _ := hedera.PrivateKeyFromString(os.Getenv("ACCOUNT_KEY"))

		// convert transaction from bytes
		transaction, err := hedera.TransactionFromBytes(body)
		if err != nil {
			log.Fatal(err)
		}

		// check the transaction type (expecting it to be TransferTransaction)
		unsignedTransaction := hedera.TransferTransaction{}

		switch t := transaction.(type) {
		case hedera.TransferTransaction:
			unsignedTransaction = t
		default:
			log.Fatal("Wrong transaction type")
		}

		// sign the transaction with private key
		signed, err := unsignedTransaction.Sign(privateKey).ToBytes()
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Transaction signed")

		// return signed transaction bytes
		fmt.Fprint(w, string(signed))
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
