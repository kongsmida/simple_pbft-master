package consensus

import (
	"crypto/sha256"
	"encoding/hex"
)

func Hash(content []byte) string {
	h := sha256.New()
	h.Write(content)
	return hex.EncodeToString(h.Sum(nil))
}

// func VRF_Prove(e string) (pk *ecdsa.PublicKey, pi []byte, r string) {
// 	//调用VRF产生随机数
// 	_, sk, err := ed25519.GenerateKey(nil)
// 	if err != nil {
// 		fmt.Println(err)
// 	}
// 	skx := secp256k1.PrivKeyFromBytes(sk)
// 	sky := skx.ToECDSA()
// 	pky := skx.PubKey().ToECDSA()
// 	randomness, pi, err := ecvrf.Secp256k1Sha256Tai.Prove(sky, []byte(hex.EncodeToString([]byte(e))))
// 	// fmt.Println(unsafe.Sizeof(randomness), unsafe.Sizeof(pi), err)
// 	if err != nil {
// 		fmt.Println(err)
// 	}
// 	return pky, pi, string(randomness)
// }

// func VRF_Verify(pk *ecdsa.PublicKey, alpha []byte, pi []byte, r []byte) (b bool) {
// 	gotb, err := ecvrf.Secp256k1Sha256Tai.Verify(pk, alpha, pi)
// 	if err != nil {
// 		fmt.Println(err)
// 	}
// 	return reflect.DeepEqual(r, gotb)
// }
