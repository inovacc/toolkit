package password

import "testing"

func TestCheckPasswordHashBcrypt(t *testing.T) {
	v, err := HashPasswordBcrypt("test")
	if err != nil {
		t.Error(err)
		return
	}

	if v == "" {
		t.Error("hash is empty")
		return
	}

	if !CheckPasswordHashBcrypt("test", v) {
		t.Error("hash not match")
		return
	}
}
