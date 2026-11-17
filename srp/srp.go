// Package srp implements the client side of the SRP-6a variant (SHA-256,
// RFC 5054 2048-bit group) used by Luanti for authentication.
//
// The construction mirrors Luanti's src/util/srp.cpp (csrp-gmp), including its
// quirks: numbers are hashed in their minimal big-endian form, except for the
// k and u values which are zero-padded to the length of N.
package srp

import (
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"math/big"
	"strings"
)

var (
	bigN = mustHex(n2048)
	bigG = big.NewInt(2)
)

const lenN = 256 // bytes in N

func mustHex(s string) *big.Int {
	n, ok := new(big.Int).SetString(s, 16)
	if !ok {
		panic("srp: bad hex constant")
	}
	return n
}

func sha(parts ...[]byte) []byte {
	h := sha256.New()
	for _, p := range parts {
		h.Write(p)
	}
	return h.Sum(nil)
}

// hashNum hashes the minimal big-endian representation of n.
func hashNum(n *big.Int) []byte { return sha(n.Bytes()) }

func pad(n *big.Int) []byte {
	b := n.Bytes()
	out := make([]byte, lenN)
	copy(out[lenN-len(b):], b)
	return out
}

func calcX(salt []byte, usernameLower string, password []byte) *big.Int {
	inner := sha([]byte(usernameLower), []byte(":"), password)
	return new(big.Int).SetBytes(sha(salt, inner))
}

// LegacyPassword converts a password into the legacy hash Luanti servers used
// before SRP (base64(sha1(name+password))). Empty passwords stay empty.
func LegacyPassword(name, password string) string {
	if password == "" {
		return ""
	}
	sum := sha1.Sum([]byte(name + password))
	return base64.StdEncoding.EncodeToString(sum[:])
}

// NewVerifier creates a random salt and the matching verifier for registering
// an account (TOSERVER_FIRST_SRP). The name is lower-cased like Luanti does.
func NewVerifier(name, password string) (salt, verifier []byte, err error) {
	salt = make([]byte, 16)
	if _, err = rand.Read(salt); err != nil {
		return nil, nil, err
	}
	return salt, VerifierWithSalt(name, password, salt), nil
}

// VerifierWithSalt computes the verifier for the given salt.
func VerifierWithSalt(name, password string, salt []byte) []byte {
	x := calcX(salt, strings.ToLower(name), []byte(password))
	return new(big.Int).Exp(bigG, x, bigN).Bytes()
}

// User holds the state of one client-side SRP exchange.
type User struct {
	name      string
	nameLower string
	password  []byte
	a, A      *big.Int
}

// NewUser starts an SRP exchange. For the legacy mechanism pass the result of
// LegacyPassword as password.
func NewUser(name, password string) (*User, error) {
	u := &User{name: name, nameLower: strings.ToLower(name), password: []byte(password)}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}
	u.a = new(big.Int).SetBytes(buf)
	u.A = new(big.Int).Exp(bigG, u.a, bigN)
	return u, nil
}

// BytesA returns the public ephemeral value A to send to the server.
func (u *User) BytesA() []byte { return u.A.Bytes() }

// ProcessChallenge consumes the server's salt and B and returns the proof M.
func (u *User) ProcessChallenge(salt, bytesB []byte) ([]byte, error) {
	B := new(big.Int).SetBytes(bytesB)
	if new(big.Int).Mod(B, bigN).Sign() == 0 {
		return nil, errors.New("srp: invalid B (safety check failed)")
	}
	if len(bytesB) > lenN {
		return nil, errors.New("srp: B too large")
	}
	uu := new(big.Int).SetBytes(sha(pad(u.A), pad(B)))
	if uu.Sign() == 0 {
		return nil, errors.New("srp: invalid u (safety check failed)")
	}
	x := calcX(salt, u.nameLower, u.password)
	k := new(big.Int).SetBytes(sha(pad(bigN), pad(bigG)))

	// S = (B - k*g^x) ^ (a + u*x) mod N
	gx := new(big.Int).Exp(bigG, x, bigN)
	base := new(big.Int).Sub(B, new(big.Int).Mul(k, gx))
	base.Mod(base, bigN)
	exp := new(big.Int).Add(u.a, new(big.Int).Mul(uu, x))
	S := new(big.Int).Exp(base, exp, bigN)
	K := hashNum(S)

	hN, hg := hashNum(bigN), hashNum(bigG)
	hxor := make([]byte, len(hN))
	for i := range hN {
		hxor[i] = hN[i] ^ hg[i]
	}
	M := sha(hxor, sha([]byte(u.name)), salt, u.A.Bytes(), B.Bytes(), K)
	return M, nil
}
