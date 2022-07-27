package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/hashgraph/hedera-sdk-go/v2"
	"github.com/joho/godotenv"
)

/*
	This application generates a transaction that transfers hbar from an external account to the operator's account
	(thus needs to be signed by the external account). In this example, the transaction for signing is sent to an external
	api server where it's signed by the external account. The signed transaction is then returned and executed by the client.

	This application only knows the operator id, operator key, and the external account id. It doesn't know the external account's private key.

	The transaction could be executed by the external api server if you don't need to do anything else with it.
*/

func init() {
	// load .env file on init
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func main() {
	// get operator id and operator key from environment variables
	operatorId, err := hedera.AccountIDFromString(os.Getenv("OPERATOR_ID"))
	if err != nil {
		log.Fatal(err)
	}

	operatorKey, err := hedera.PrivateKeyFromString(os.Getenv("OPERATOR_KEY"))
	if err != nil {
		log.Fatal(err)
	}

	// create client
	client := hedera.ClientForTestnet()
	client.SetOperator(operatorId, operatorKey)

	// external account id that needs to sign the transaction
	externalAccountId, err := hedera.AccountIDFromString("0.0.46809373")
	if err != nil {
		log.Fatal(err)
	}

	/*
		Manually generate and set the transaction id using the external account id. This will make the external account cover the transaction fees.
		Alternatively you can set it to the operator account id if you wish to cover the transaction fees.
	*/

	// generate a transaction id using the external account id
	tid := hedera.TransactionIDGenerate(externalAccountId)

	// create a new transaction transferring hbar from the external account to our operator (requires external account to sign)
	transactionToSign, err := hedera.NewTransferTransaction().
		SetTransactionID(tid).
		AddHbarTransfer(externalAccountId, hedera.HbarFrom(-100000000, hedera.HbarUnits.Tinybar)).
		AddHbarTransfer(operatorId, hedera.HbarFrom(100000000, hedera.HbarUnits.Tinybar)).
		SetTransactionMemo("signed externally").
		FreezeWith(client)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Transaction generated and frozen")

	// convert the transaction to bytes
	transactionToSignBytes, err := transactionToSign.ToBytes()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Transaction converted to bytes")

	// send the transaction to external signing service
	signedTransactionBytes, err := signingService(transactionToSignBytes)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Received signed transaction")

	// convert the signed transaction from bytes
	signedTransaction, err := hedera.TransactionFromBytes(signedTransactionBytes)
	if err != nil {
		log.Fatal(err)
	}

	// check the transaction type (expecting it to be TransferTransaction)
	transactionToSubmit := hedera.TransferTransaction{}

	switch t := signedTransaction.(type) {
	case hedera.TransferTransaction:
		transactionToSubmit = t
	default:
		log.Fatal("Wrong transaction type")
	}

	fmt.Println("Executing transaction")
	// execute the transaction with client
	resp, err := transactionToSubmit.Execute(client)
	if err != nil {
		log.Fatal(err)
	}

	// get the receipt with client
	receipt, err := resp.GetReceipt(client)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Transaction result:", receipt.Status.String())
}

func signingService(t []byte) ([]byte, error) {
	fmt.Println("Sending transaction for signing")

	// make a request to the external api with the transaction bytes
	response, err := http.Post(os.Getenv("EXTERNAL_API_URL"), "", bytes.NewBuffer(t))
	if err != nil {
		return []byte{}, err
	}
	defer response.Body.Close()

	// read the response and return for execution
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return []byte{}, err
	}

	return body, nil
}
