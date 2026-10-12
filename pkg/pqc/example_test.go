package pqc_test

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"

	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/fbsobreira/gotron-sdk/pkg/pqc"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
)

// Generate an ML-DSA-44 key and derive its TRON address. ML-DSA-44 is pure Go,
// so this works in every build.
func ExampleGenerateKey() {
	key, err := pqc.GenerateKey(core.PQScheme_ML_DSA_44, rand.Reader)
	if err != nil {
		log.Fatal(err)
	}
	defer key.Destroy()

	addr := key.Public().Address().String()
	fmt.Println(key.Scheme())
	fmt.Println(addr[0] == 'T')

	// Output:
	// ML_DSA_44
	// true
}

// Save a key as a wallet-cli compatible keystore and load it back.
func ExampleEncryptKeystore() {
	key, err := pqc.GenerateKey(core.PQScheme_ML_DSA_44, rand.Reader)
	if err != nil {
		log.Fatal(err)
	}
	defer key.Destroy()

	// LightScryptN keeps this example fast. For a real keystore use
	// keystore.StandardScryptN and keystore.StandardScryptP.
	data, err := pqc.EncryptKeystore(key, "example passphrase", keystore.LightScryptN, keystore.LightScryptP)
	if err != nil {
		log.Fatal(err)
	}

	loaded, err := pqc.DecryptKeystore(data, "example passphrase")
	if err != nil {
		log.Fatal(err)
	}
	defer loaded.Destroy()

	fmt.Println(loaded.Public().Equal(key.Public()))

	// Output:
	// true
}

// Export a key as java-tron Toolkit JSON and parse it back. The file holds the
// private key in clear, so protect it like the key itself.
func ExampleMarshalToolkitJSON() {
	key, err := pqc.GenerateKey(core.PQScheme_ML_DSA_44, rand.Reader)
	if err != nil {
		log.Fatal(err)
	}
	defer key.Destroy()

	data, err := pqc.MarshalToolkitJSON(key)
	if err != nil {
		log.Fatal(err)
	}

	parsed, err := pqc.ParseToolkitJSON(data)
	if err != nil {
		log.Fatal(err)
	}
	defer parsed.Destroy()

	fmt.Println(parsed.Public().Equal(key.Public()))

	// Output:
	// true
}

// Check whether this build can sign with a scheme. FN-DSA-512 needs the falcon
// build tag, CGO and liboqs; ML-DSA-44 is always available.
func ExampleSchemeAvailable() {
	fmt.Println(pqc.SchemeAvailable(core.PQScheme_ML_DSA_44) == nil)

	if err := pqc.SchemeAvailable(core.PQScheme_FN_DSA_512); errors.Is(err, pqc.ErrFalconUnavailable) {
		fmt.Println("FN-DSA-512 signing is not built in; use ML-DSA-44 or build with -tags falcon")
	} else if err != nil {
		log.Fatal(err)
	}
}
