package nixb32

import (
	"encoding/hex"
	"testing"
)

func TestHash(t *testing.T) {
	expected := "0ysj00x31q08vxsznqd9pmvwa0rrzza8qqjy3hcvhallzm054cxb"

	in := "ab335240fd942ab8191c5e628cd4ff3903c577bda961fb75df08e0303a00527b"
	b, err := hex.DecodeString(in)
	if err != nil {
		panic(err)
	}
	x, err := Hash(b)
	if err != nil {
		panic(err)
	}
	if x != expected {
		t.Fatalf("actual=%s expected=%s", x, expected)
	}
}
