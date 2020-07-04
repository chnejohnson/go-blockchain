package wallet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"log"

	"golang.org/x/crypto/ripemd160"
)

const (
	checksumLength = 4
	version        = byte(0x00)
)

// Wallet contains private key and public key
type Wallet struct {
	PrivateKey ecdsa.PrivateKey
	PublicKey  []byte
}

// Address generate the address of the wallet
func (w Wallet) Address() []byte {
	pubKeyHash := PublicKeyHash(w.PublicKey)

	versionedHash := append([]byte{version}, pubKeyHash...)
	checksum := Checksum(versionedHash)

	fullHash := append(versionedHash, checksum...)
	address := Base58Encode(fullHash)

	// fmt.Printf("pub key: %x\n", w.PublicKey)
	// fmt.Printf("pub key hash: %x\n", pubKeyHash)
	// fmt.Printf("full hash: %x\n", fullHash)
	// fmt.Printf("address: %s\n", address)

	return address

}

// NewKeyPair generate private key and public key
func NewKeyPair() (ecdsa.PrivateKey, []byte) {
	curve := elliptic.P256()
	private, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		log.Panic(err)
	}

	pub := append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)
	return *private, pub
}

// MakeWallet generate a wallet with a pair of key
func MakeWallet() *Wallet {
	private, public := NewKeyPair()
	wallet := Wallet{private, public}
	return &wallet
}

// PublicKeyHash generate public key hash with public key
func PublicKeyHash(pubKey []byte) []byte {
	pubHash := sha256.Sum256(pubKey)

	hasher := ripemd160.New()
	_, err := hasher.Write(pubHash[:])
	if err != nil {
		log.Panic(err)
	}

	publicRipMD := hasher.Sum(nil)
	return publicRipMD
}

// Checksum generate 4 bytes checksum with payload by sha256 twice
func Checksum(payload []byte) []byte {
	firstHash := sha256.Sum256(payload)
	secondHash := sha256.Sum256(firstHash[:])

	return secondHash[:checksumLength]
}
