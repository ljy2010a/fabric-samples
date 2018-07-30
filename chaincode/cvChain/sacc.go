/*
 * Copyright IBM Corp All Rights Reserved
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package main
//
//import (
//	"fmt"
//
//	"github.com/hyperledger/fabric/core/chaincode/shim"
//	"github.com/hyperledger/fabric/protos/peer"
//	"encoding/json"
//)
//
//// SimpleAsset implements a simple chaincode to manage an asset
//type SimpleAsset struct {
//}
//
//// Init is called during chaincode instantiation to initialize any
//// data. Note that chaincode upgrade also calls this function to reset
//// or to migrate data.
//func (t *SimpleAsset) Init(stub shim.ChaincodeStubInterface) peer.Response {
//	// Get the args from the transaction proposal
//	args := stub.GetStringArgs()
//	if len(args) != 2 {
//		return shim.Error("Incorrect arguments. Expecting a key and a value")
//	}
//
//	// Set up any variables or assets here by calling stub.PutState()
//
//	// We store the key and the value on the ledger
//	err := stub.PutState(args[0], []byte(args[1]))
//	if err != nil {
//		return shim.Error(fmt.Sprintf("Failed to create asset: %s", args[0]))
//	}
//	return shim.Success(nil)
//}
//
//// Invoke is called per transaction on the chaincode. Each transaction is
//// either a 'get' or a 'set' on the asset created by Init function. The Set
//// method may create a new asset by specifying a new key-value pair.
//func (t *SimpleAsset) Invoke(stub shim.ChaincodeStubInterface) peer.Response {
//	// Extract the function and args from the transaction proposal
//	fn, args := stub.GetFunctionAndParameters()
//
//	var result string
//	var err error
//
//	if fn == "addRecord" {
//		result, err = addRecord(stub, args)
//	}
//
//	if fn == "getRecord" {
//		result, err = getRecord(stub, args)
//	}
//
//	if err != nil {
//		return shim.Error(err.Error())
//	}
//
//	// Return the result as success payload
//	return shim.Success([]byte(result))
//}
//
//// Set stores the asset (both key and value) on the ledger. If the key exists,
//// it will override the value with the new one
//func addRecord(stub shim.ChaincodeStubInterface, args []string) (string, error) {
//	if len(args) != 4 {
//		return "", fmt.Errorf("Incorrect arguments. Expecting a key and a value")
//	}
//
//	key := fmt.Sprintf("%s-%s", args[0], args[1])
//	valueMap := make(map[string]string)
//	valueMap["company"] = args[2]
//	valueMap["job"] = args[3]
//	value, _ := json.Marshal(valueMap)
//
//	err := stub.PutState(key, value)
//	if err != nil {
//		return "", fmt.Errorf("Failed to set asset: %s", args[0])
//	}
//	return args[1], nil
//}
//
//// Get returns the value of the specified asset key
//func getRecord(stub shim.ChaincodeStubInterface, args []string) (string, error) {
//	if len(args) != 2 {
//		return "", fmt.Errorf("Incorrect arguments. Expecting a key")
//	}
//
//	key := fmt.Sprintf("%s-%s", args[0], args[1])
//	value, err := stub.GetState(key)
//	if err != nil {
//		return "", fmt.Errorf("Failed to get asset: %s with error: %s", args[0], err)
//	}
//	if value == nil {
//		return "", fmt.Errorf("Asset not found: %s", args[0])
//	}
//	valueMap := make(map[string]string)
//	err = json.Unmarshal(value, valueMap)
//	if err != nil {
//		return "", fmt.Errorf("Failed to parse valuse: %s ", string(value))
//	}
//
//	if company, ok := valueMap["valueMap"]; ok {
//		return company, nil
//	}
//	return "", fmt.Errorf("valuse get fail : %s ", string(value))
//}
//
//// main function starts up the chaincode in the container during instantiate
//func main() {
//	if err := shim.Start(new(SimpleAsset)); err != nil {
//		fmt.Printf("Error starting SimpleAsset chaincode: %s", err)
//	}
//}
