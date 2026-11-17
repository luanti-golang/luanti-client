package srp

import (
	"bytes"
	"math/big"
	"testing"
)

// serverM is a tiny server-side SRP-6a implementation following Luanti's
// srp_verifier_new, used to check both sides compute the same proof.
func serverM(name string, salt, verifier, bytesA []byte) (B, M []byte) {
	v := new(big.Int).SetBytes(verifier)
	A := new(big.Int).SetBytes(bytesA)
	b := big.NewInt(0xdeadbeefcafe)
	k := new(big.Int).SetBytes(sha(pad(bigN), pad(bigG)))
	Bn := new(big.Int).Add(new(big.Int).Mul(k, v), new(big.Int).Exp(bigG, b, bigN))
	Bn.Mod(Bn, bigN)
	u := new(big.Int).SetBytes(sha(pad(A), pad(Bn)))
	// S = (A * v^u) ^ b
	S := new(big.Int).Exp(new(big.Int).Mul(A, new(big.Int).Exp(v, u, bigN)), b, bigN)
	K := hashNum(S)
	hN, hg := hashNum(bigN), hashNum(bigG)
	hx := make([]byte, len(hN))
	for i := range hN {
		hx[i] = hN[i] ^ hg[i]
	}
	return Bn.Bytes(), sha(hx, sha([]byte(name)), salt, A.Bytes(), Bn.Bytes(), K)
}

func TestRoundTrip(t *testing.T) {
	for _, pw := range []string{"hunter2", "", "ünïcode pass"} {
		salt, v, err := NewVerifier("Alice", pw)
		if err != nil {
			t.Fatal(err)
		}
		u, err := NewUser("Alice", pw)
		if err != nil {
			t.Fatal(err)
		}
		B, wantM := serverM("Alice", salt, v, u.BytesA())
		gotM, err := u.ProcessChallenge(salt, B)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(gotM, wantM) {
			t.Fatalf("pw %q: proofs differ", pw)
		}
	}
}

func TestWrongPassword(t *testing.T) {
	salt, v, _ := NewVerifier("Bob", "right")
	u, _ := NewUser("Bob", "wrong")
	B, wantM := serverM("Bob", salt, v, u.BytesA())
	gotM, _ := u.ProcessChallenge(salt, B)
	if bytes.Equal(gotM, wantM) {
		t.Fatal("wrong password produced a valid proof")
	}
}

func TestRejectZeroB(t *testing.T) {
	u, _ := NewUser("x", "y")
	if _, err := u.ProcessChallenge([]byte("salt"), []byte{0}); err == nil {
		t.Fatal("expected error for B=0")
	}
}
