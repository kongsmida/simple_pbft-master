package main

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"os"
	"reflect"

	"simple_pbft-master/pbft/network"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/vechain/go-ecvrf"
)

func VRF_Prove(e string) (pk *ecdsa.PublicKey, pi []byte, r []byte) {
	//调用VRF产生随机数
	_, sk, err := ed25519.GenerateKey(nil)
	if err != nil {
		fmt.Println(err)
	}
	skx := secp256k1.PrivKeyFromBytes(sk)
	sky := skx.ToECDSA()
	pky := skx.PubKey().ToECDSA()
	randomness, pi, err := ecvrf.Secp256k1Sha256Tai.Prove(sky, []byte(hex.EncodeToString([]byte(e))))
	// fmt.Println(unsafe.Sizeof(randomness), unsafe.Sizeof(pi), err)
	if err != nil {
		fmt.Println("proverr", err)
	}
	return pky, pi, randomness
}

func VRF_Verify(pk *ecdsa.PublicKey, alpha []byte, pi []byte, r []byte) (b bool) {
	gotb, err := ecvrf.Secp256k1Sha256Tai.Verify(pk, alpha, pi)
	if err != nil {
		fmt.Println("verifyerr", err)
	}
	return reflect.DeepEqual(r, gotb)
}

func main() {
	// var epoch string = time.Now().String()
	// pky, pi, randomness := VRF_Prove(epoch)
	// fmt.Println(VRF_Verify(pky, []byte(epoch), pi, randomness))

	nodeID := os.Args[1]
	server := network.NewServer(nodeID)
	server.Start()
}
