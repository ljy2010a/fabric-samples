/*
Copyright IBM Corp. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"fmt"

	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/core/chaincode/shim/ext/entities"
	pb "github.com/hyperledger/fabric/protos/peer"
	"encoding/json"
	"github.com/pkg/errors"
)

// CORE_PEER_ADDRESS=peer:7052 CORE_CHAINCODE_ID_NAME=cvcc:0 ./cvChain
// peer chaincode install -p chaincodedev/chaincode/cvChain -n cvcc -v 0 && peer chaincode instantiate -n cvcc -v 0 -C myc -c '{"Args":[]}'
// ENCKEY=`openssl rand 32 -base64` && DECKEY=$ENCKEY && IV=`openssl rand 16 -base64`
// peer chaincode invoke -n cvcc -C myc -c '{"Args":["encRecord","1009","2006","corp2","engineer"]}' --transient "{\"ENCKEY\":\"$ENCKEY\",\"IV\":\"$IV\"}"
// peer chaincode query -n cvcc -C myc -c '{"Args":["decRecord","1009","2006"]}' --transient "{\"DECKEY\":\"$DECKEY\"}"

// peer chaincode invoke -n cvcc -C myc -c '{"Args":["addRecord","1010","2006","okok2","engineer"]}'
// peer chaincode query -n cvcc -C myc -c '{"Args":["getRecord","1010","2006"]}'

// the functions below show some best practices on how
// to use entities to perform cryptographic operations
// over the ledger state

// getStateAndDecrypt retrieves the value associated to key,
// decrypts it with the supplied entity and returns the result
// of the decryption
func getStateAndDecrypt(stub shim.ChaincodeStubInterface, ent entities.Encrypter, key string) ([]byte, error) {
	// at first we retrieve the ciphertext from the ledger
	ciphertext, err := stub.GetState(key)
	if err != nil {
		return nil, err
	}

	// GetState will return a nil slice if the key does not exist.
	// Note that the chaincode logic may want to distinguish between
	// nil slice (key doesn't exist in state db) and empty slice
	// (key found in state db but value is empty). We do not
	// distinguish the case here
	if len(ciphertext) == 0 {
		return nil, errors.New("no ciphertext to decrypt")
	}

	return ent.Decrypt(ciphertext)
}

// encryptAndPutState encrypts the supplied value using the
// supplied entity and puts it to the ledger associated to
// the supplied KVS key
func encryptAndPutState(stub shim.ChaincodeStubInterface, ent entities.Encrypter, key string, value []byte) error {
	// at first we use the supplied entity to encrypt the value
	ciphertext, err := ent.Encrypt(value)
	if err != nil {
		return err
	}

	return stub.PutState(key, ciphertext)
}

// getStateDecryptAndVerify retrieves the value associated to key,
// decrypts it with the supplied entity, verifies the signature
// over it and returns the result of the decryption in case of
// success
func getStateDecryptAndVerify(stub shim.ChaincodeStubInterface, ent entities.EncrypterSignerEntity, key string) ([]byte, error) {
	// here we retrieve and decrypt the state associated to key
	val, err := getStateAndDecrypt(stub, ent, key)
	if err != nil {
		return nil, err
	}

	// we unmarshal a SignedMessage from the decrypted state
	msg := &entities.SignedMessage{}
	err = msg.FromBytes(val)
	if err != nil {
		return nil, err
	}

	// we verify the signature
	ok, err := msg.Verify(ent)
	if err != nil {
		return nil, err
	} else if !ok {
		return nil, errors.New("invalid signature")
	}

	return msg.Payload, nil
}

// signEncryptAndPutState signs the supplied value, encrypts
// the supplied value together with its signature using the
// supplied entity and puts it to the ledger associated to
// the supplied KVS key
func signEncryptAndPutState(stub shim.ChaincodeStubInterface, ent entities.EncrypterSignerEntity, key string, value []byte) error {
	// here we create a SignedMessage, set its payload
	// to value and the ID of the entity and
	// sign it with the entity
	msg := &entities.SignedMessage{Payload: value, ID: []byte(ent.ID())}
	err := msg.Sign(ent)
	if err != nil {
		return err
	}

	// here we serialize the SignedMessage
	b, err := msg.ToBytes()
	if err != nil {
		return err
	}

	// here we encrypt the serialized version associated to args[0]
	return encryptAndPutState(stub, ent, key, b)
}

type keyValuePair struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// getStateByRangeAndDecrypt retrieves a range of KVS pairs from the
// ledger and decrypts each value with the supplied entity; it returns
// a json-marshalled slice of keyValuePair
func getStateByRangeAndDecrypt(stub shim.ChaincodeStubInterface, ent entities.Encrypter, startKey, endKey string) ([]byte, error) {
	// we call get state by range to go through the entire range
	iterator, err := stub.GetStateByRange(startKey, endKey)
	if err != nil {
		return nil, err
	}
	defer iterator.Close()

	// we decrypt each entry - the assumption is that they have all been encrypted with the same key
	keyvalueset := []keyValuePair{}
	for iterator.HasNext() {
		el, err := iterator.Next()
		if err != nil {
			return nil, err
		}

		v, err := ent.Decrypt(el.Value)
		if err != nil {
			return nil, err
		}

		keyvalueset = append(keyvalueset, keyValuePair{el.Key, string(v)})
	}

	bytes, err := json.Marshal(keyvalueset)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

const DECKEY = "DECKEY"
const VERKEY = "VERKEY"
const ENCKEY = "ENCKEY"
const SIGKEY = "SIGKEY"
const IV = "IV"

func genKey(args []string) string {
	return fmt.Sprintf("%s-%s", args[0], args[1])
}

func genKeyAndValue(args []string) (key string, value []byte) {
	key = genKey(args)
	value = []byte(args[2])
	//valueMap := make(map[string]string)
	//valueMap["company"] = args[2]
	//valueMap["job"] = args[3]
	//value, _ = json.Marshal(valueMap)
	return
}

func ParseValueCompany(value []byte) (company string, err error) {
	valueMap := make(map[string]string)
	err = json.Unmarshal(value, valueMap)
	if err != nil {
		return "", fmt.Errorf("Failed to parse valuse: %s ", string(value))
	}

	if company, ok := valueMap["company"]; ok {
		return company, nil
	}
	return "", fmt.Errorf("valuse get fail : %s ", string(value))
}

// EncCC example simple Chaincode implementation of a chaincode that uses encryption/signatures
type EncCC struct {
	bccspInst bccsp.BCCSP
}

// Init does nothing for this cc
func (t *EncCC) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

// Encrypter exposes how to write state to the ledger after having
// encrypted it with an AES 256 bit key that has been provided to the chaincode through the
// transient field
func (t *EncCC) Encrypter(stub shim.ChaincodeStubInterface, args []string, encKey, IV []byte) pb.Response {
	// create the encrypter entity - we give it an ID, the bccsp instance, the key and (optionally) the IV
	ent, err := entities.NewAES256EncrypterEntity("ID", t.bccspInst, encKey, IV)
	if err != nil {
		return shim.Error(fmt.Sprintf("entities.NewAES256EncrypterEntity failed, err %s", err))
	}

	if len(args) != 4 {
		return shim.Error("Expected 4 parameters to function Encrypter")
	}

	key, value := genKeyAndValue(args)

	// here, we encrypt cleartextValue and assign it to key
	err = encryptAndPutState(stub, ent, key, value)
	if err != nil {
		return shim.Error(fmt.Sprintf("encryptAndPutState failed, err %+v", err))
	}
	return shim.Success([]byte(key))
}

// Decrypter exposes how to read from the ledger and decrypt using an AES 256
// bit key that has been provided to the chaincode through the transient field.
func (t *EncCC) Decrypter(stub shim.ChaincodeStubInterface, args []string, decKey, IV []byte) pb.Response {
	// create the encrypter entity - we give it an ID, the bccsp instance, the key and (optionally) the IV
	ent, err := entities.NewAES256EncrypterEntity("ID", t.bccspInst, decKey, IV)
	if err != nil {
		return shim.Error(fmt.Sprintf("entities.NewAES256EncrypterEntity failed, err %s", err))
	}

	if len(args) != 2 {
		return shim.Error("Expected 2 parameters to function Decrypter")
	}

	key := genKey(args)

	// here we decrypt the state associated to key
	cleartextValue, err := getStateAndDecrypt(stub, ent, key)
	if err != nil {
		return shim.Error(fmt.Sprintf("getStateAndDecrypt failed, err %+v", err))
	}
	return shim.Success([]byte(cleartextValue))

	//company, err := ParseValueCompany(cleartextValue)
	//if err != nil {
	//	return shim.Error(fmt.Sprintf("ParseValueCompany failed, err %+v", err))
	//}
	//
	//// here we return the decrypted value as a result
	//return shim.Success([]byte(cleartextValue))
}

// EncrypterSigner exposes how to write state to the ledger after having received keys for
// encrypting (AES 256 bit key) and signing (X9.62/SECG curve over a 256 bit prime field) that has been provided to the chaincode through the
// transient field
func (t *EncCC) EncrypterSigner(stub shim.ChaincodeStubInterface, args []string, encKey, sigKey []byte) pb.Response {
	// create the encrypter/signer entity - we give it an ID, the bccsp instance and the keys
	ent, err := entities.NewAES256EncrypterECDSASignerEntity("ID", t.bccspInst, encKey, sigKey)
	if err != nil {
		return shim.Error(fmt.Sprintf("entities.NewAES256EncrypterEntity failed, err %s", err))
	}

	if len(args) != 2 {
		return shim.Error("Expected 2 parameters to function EncrypterSigner")
	}

	key := args[0]
	cleartextValue := []byte(args[1])

	// here, we sign cleartextValue, encrypt it and assign it to key
	err = signEncryptAndPutState(stub, ent, key, cleartextValue)
	if err != nil {
		return shim.Error(fmt.Sprintf("signEncryptAndPutState failed, err %+v", err))
	}

	return shim.Success(nil)
}

// DecrypterVerify exposes how to get state to the ledger after having received keys for
// decrypting (AES 256 bit key) and verifying (X9.62/SECG curve over a 256 bit prime field) that has been provided to the chaincode through the
// transient field
func (t *EncCC) DecrypterVerify(stub shim.ChaincodeStubInterface, args []string, decKey, verKey []byte) pb.Response {
	// create the decrypter/verify entity - we give it an ID, the bccsp instance and the keys
	ent, err := entities.NewAES256EncrypterECDSASignerEntity("ID", t.bccspInst, decKey, verKey)
	if err != nil {
		return shim.Error(fmt.Sprintf("entities.NewAES256DecrypterEntity failed, err %s", err))
	}

	if len(args) != 1 {
		return shim.Error("Expected 1 parameters to function DecrypterVerify")
	}
	key := args[0]

	// here we decrypt the state associated to key and verify it
	cleartextValue, err := getStateDecryptAndVerify(stub, ent, key)
	if err != nil {
		return shim.Error(fmt.Sprintf("getStateDecryptAndVerify failed, err %+v", err))
	}

	// here we return the decrypted and verified value as a result
	return shim.Success(cleartextValue)
}

// RangeDecrypter shows how range queries may be satisfied by using the decrypter
// entity directly to decrypt previously encrypted key-value pairs
func (t *EncCC) RangeDecrypter(stub shim.ChaincodeStubInterface, decKey []byte) pb.Response {
	// create the encrypter entity - we give it an ID, the bccsp instance and the key
	ent, err := entities.NewAES256EncrypterEntity("ID", t.bccspInst, decKey, nil)
	if err != nil {
		return shim.Error(fmt.Sprintf("entities.NewAES256EncrypterEntity failed, err %s", err))
	}

	bytes, err := getStateByRangeAndDecrypt(stub, ent, "", "")
	if err != nil {
		return shim.Error(fmt.Sprintf("getStateByRangeAndDecrypt failed, err %+v", err))
	}

	return shim.Success(bytes)
}

// Set stores the asset (both key and value) on the ledger. If the key exists,
// it will override the value with the new one
func addRecord(stub shim.ChaincodeStubInterface, args []string) (string, error) {
	if len(args) != 4 {
		return "", fmt.Errorf("Incorrect arguments. Expecting a key and a value")
	}

	key, value := genKeyAndValue(args)

	err := stub.PutState(key, value)
	if err != nil {
		return "", fmt.Errorf("Failed to set asset: %s", args[0])
	}
	return key, nil
}

// Get returns the value of the specified asset key
func getRecord(stub shim.ChaincodeStubInterface, args []string) (string, error) {
	if len(args) != 2 {
		return "", fmt.Errorf("Incorrect arguments. Expecting a key")
	}

	key := genKey(args)
	value, err := stub.GetState(key)
	if err != nil {
		return "", fmt.Errorf("Failed to get asset: %s with error: %s", args[0], err)
	}
	if value == nil {
		return "", fmt.Errorf("Asset not found: %s", args[0])
	}
	return string(value), nil

	//valueMap := make(map[string]string)
	//err = json.Unmarshal(value, valueMap)
	//if err != nil {
	//	return "", fmt.Errorf("Failed to parse valuse: %s ", string(value))
	//}
	//
	//if company, ok := valueMap["company"]; ok {
	//	return company, nil
	//}
	//return "", fmt.Errorf("valuse get fail : %s ", string(value))
}

// Invoke for this chaincode exposes functions to ENCRYPT, DECRYPT transactional
// data.  It also supports an example to ENCRYPT and SIGN and DECRYPT and
// VERIFY.  The Initialization Vector (IV) can be passed in as a parm to
// ensure peers have deterministic data.
func (t *EncCC) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	// get arguments and transient
	f, args := stub.GetFunctionAndParameters()

	switch f {
	case "encRecord":
		tMap, err := stub.GetTransient()
		if err != nil {
			return shim.Error(fmt.Sprintf("Could not retrieve transient, err %s", err))
		}
		// make sure there's a key in transient - the assumption is that
		// it's associated to the string "ENCKEY"
		if _, in := tMap[ENCKEY]; !in {
			return shim.Error(fmt.Sprintf("Expected transient encryption key %s", ENCKEY))
		}

		return t.Encrypter(stub, args[0:], tMap[ENCKEY], tMap[IV])
	case "decRecord":
		tMap, err := stub.GetTransient()
		if err != nil {
			return shim.Error(fmt.Sprintf("Could not retrieve transient, err %s", err))
		}

		// make sure there's a key in transient - the assumption is that
		// it's associated to the string "DECKEY"
		if _, in := tMap[DECKEY]; !in {
			return shim.Error(fmt.Sprintf("Expected transient decryption key %s", DECKEY))
		}

		return t.Decrypter(stub, args[0:], tMap[DECKEY], tMap[IV])

	case "ENCRYPTSIGN":
		tMap, err := stub.GetTransient()
		if err != nil {
			return shim.Error(fmt.Sprintf("Could not retrieve transient, err %s", err))
		}
		// make sure keys are there in the transient map - the assumption is that they
		// are associated to the string "ENCKEY" and "SIGKEY"
		if _, in := tMap[ENCKEY]; !in {
			return shim.Error(fmt.Sprintf("Expected transient key %s", ENCKEY))
		} else if _, in := tMap[SIGKEY]; !in {
			return shim.Error(fmt.Sprintf("Expected transient key %s", SIGKEY))
		}

		return t.EncrypterSigner(stub, args[0:], tMap[ENCKEY], tMap[SIGKEY])
	case "DECRYPTVERIFY":
		tMap, err := stub.GetTransient()
		if err != nil {
			return shim.Error(fmt.Sprintf("Could not retrieve transient, err %s", err))
		}
		// make sure keys are there in the transient map - the assumption is that they
		// are associated to the string "DECKEY" and "VERKEY"
		if _, in := tMap[DECKEY]; !in {
			return shim.Error(fmt.Sprintf("Expected transient key %s", DECKEY))
		} else if _, in := tMap[VERKEY]; !in {
			return shim.Error(fmt.Sprintf("Expected transient key %s", VERKEY))
		}

		return t.DecrypterVerify(stub, args[0:], tMap[DECKEY], tMap[VERKEY])
	case "RANGEQUERY":
		tMap, err := stub.GetTransient()
		if err != nil {
			return shim.Error(fmt.Sprintf("Could not retrieve transient, err %s", err))
		}
		// make sure there's a key in transient - the assumption is that
		// it's associated to the string "ENCKEY"
		if _, in := tMap[DECKEY]; !in {
			return shim.Error(fmt.Sprintf("Expected transient key %s", DECKEY))
		}

		return t.RangeDecrypter(stub, tMap[DECKEY])

	case "addRecord":
		result, err := addRecord(stub, args)
		if err != nil {
			return shim.Error(err.Error())
		}
		return shim.Success([]byte(result))

	case "getRecord":
		result, err := getRecord(stub, args)
		if err != nil {
			return shim.Error(err.Error())
		}
		return shim.Success([]byte(result))

	default:
		return shim.Error(fmt.Sprintf("Unsupported function %s", f))
	}
}

func main() {
	factory.InitFactories(nil)

	err := shim.Start(&EncCC{factory.GetDefault()})
	if err != nil {
		fmt.Printf("Error starting EncCC chaincode: %s", err)
	}
}
