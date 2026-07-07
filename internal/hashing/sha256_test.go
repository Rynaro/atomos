package hashing

import "testing"

func TestSHA256HexMatchesShasumConvention(t *testing.T) {
	// echo -n "" | shasum -a 256
	if got, want := SHA256Hex(nil), "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"; got != want {
		t.Errorf("SHA256Hex(nil) = %q, want %q", got, want)
	}
	// echo -n "hello" | shasum -a 256
	if got, want := SHA256HexString("hello"), "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"; got != want {
		t.Errorf("SHA256HexString(hello) = %q, want %q", got, want)
	}
}
