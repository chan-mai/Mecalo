package password

import "testing"

func TestHashVerify(t *testing.T) {
	h := Fast()
	hash, err := h.Hash("correct horse")
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := h.Verify(hash, "correct horse"); err != nil || !ok {
		t.Errorf("verify = %v, %v", ok, err)
	}
	if ok, _ := h.Verify(hash, "wrong"); ok {
		t.Error("wrong password accepted")
	}
	if _, err := h.Verify("plain", "x"); err == nil {
		t.Error("malformed hash accepted")
	}
}
