package base58

import (
	"fmt"
	"log"
)

// This Example shows the basic usage of the package:
// Encode, Decode
func Example_basic() {
	// use bitcoin alphabet, just same as: base58.NewAlphabet("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")
	myAlphabet := BitcoinAlphabet

	// encode
	input := []byte{0, 0, 0, 1, 2, 3}
	encodedString := Encode(input, myAlphabet)
	fmt.Printf("base58encode(%v) = %s\n", input, encodedString)

	// decode
	decodedBytes, err := Decode(encodedString, myAlphabet)
	if err != nil { // error occurred when encodedString contains character not in alphabet
		log.Fatal("base58Decode error:", err)
	} else {
		fmt.Printf("base58decode(%s) = %v\n", encodedString, decodedBytes)
	}

	// Output:
	// base58encode([0 0 0 1 2 3]) = 111Ldp
	// base58decode(111Ldp) = [0 0 0 1 2 3]
}
