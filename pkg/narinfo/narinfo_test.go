package narinfo

import (
	"reflect"
	"strings"
	"testing"
)

const testPath string = "/nix/store/a5546jp1hl164cklp4271rjavgacn0p7-hello-2.10"
const testPubKey string = "test.example.org:sB+dVaXA1LUlVCZ1kqsxtL6XMODrDzenNUETvW/5yEw="
const testPrivKey string = "test.example.org:iGQhaAlkDi7KaSdS8wEBwdWjYSEpUXVqbbIrCt49tXiwH51VpcDUtSVUJnWSqzG0vpcw4OsPN6c1QRO9b/nITA=="
const testSig string = "test.example.org:cPS0OhXDKFgRogK3IipBVDfYa/Dd3j0N8JCx8lu4UYjqMsdZIBntXAvwoSN4iHMGA1jbO0oQA1iSBHa4GDDyDw=="

func getUnsignedValidNarInfo() NarInfo {
	//	# this is github:nixos/nixpkgs/2deeb58f49480f468adca6b08291322de4dbce6b#hello
	//	❯ curl 'https://cache.nixos.org/a5546jp1hl164cklp4271rjavgacn0p7.narinfo'
	//	StorePath: /nix/store/a5546jp1hl164cklp4271rjavgacn0p7-hello-2.10
	//	URL: nar/13l46s8q9dpjkny054p2bqgrc78hapk6k5gllsk97ajcn9iycrr4.nar.xz
	//	Compression: xz
	//	FileHash: sha256:13l46s8q9dpjkny054p2bqgrc78hapk6k5gllsk97ajcn9iycrr4
	//	FileSize: 41168
	//	NarHash: sha256:1k84a1cym428dfaa8z0lzl64cb3d6cf5cl8lck9ifzimzjz1hhm8
	//	NarSize: 206000
	//	References: 2wrfwfdpklhaqhjxgq6yd257cagdxgph-glibc-2.32 a5546jp1hl164cklp4271rjavgacn0p7-hello-2.10
	//	Deriver: vnny8nvl2x7wrhgc616qdf94hhspm0vn-hello-2.10.drv
	//	Sig: cache.nixos.org-1:aqWM1rLFL8bsLPjx3NnR55ARAfmoKaUGDhiD5Nzve7slQYUZloRdSuOmwIdC7RckeHzzdwufS4GRbMhK9mt4BA==

	return NarInfo{
		StorePath: testPath,
		URL:       "nar/13l46s8q9dpjkny054p2bqgrc78hapk6k5gllsk97ajcn9iycrr4.nar.xz",
		FileHash:  "sha256:13l46s8q9dpjkny054p2bqgrc78hapk6k5gllsk97ajcn9iycrr4",
		FileSize:  41168,
		NarHash:   "sha256:1k84a1cym428dfaa8z0lzl64cb3d6cf5cl8lck9ifzimzjz1hhm8",
		NarSize:   206000,
		References: []string{
			"2wrfwfdpklhaqhjxgq6yd257cagdxgph-glibc-2.32",
			"a5546jp1hl164cklp4271rjavgacn0p7-hello-2.10",
		},
		Deriver:    "vnny8nvl2x7wrhgc616qdf94hhspm0vn-hello-2.10.drv",
		Signatures: []string{},
	}
}

// this works in spite of the sloppy narinfo...
func TestFingerprint(t *testing.T) {
	ni := getUnsignedValidNarInfo()
	expectedFP := []byte("1;/nix/store/a5546jp1hl164cklp4271rjavgacn0p7-hello-2.10" +
		";sha256:1k84a1cym428dfaa8z0lzl64cb3d6cf5cl8lck9ifzimzjz1hhm8;206000" +
		";/nix/store/2wrfwfdpklhaqhjxgq6yd257cagdxgph-glibc-2.32,/nix/store/a5546jp1hl164cklp4271rjavgacn0p7-hello-2.10")
	actualFP, err := ni.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(actualFP, expectedFP) {
		t.Fatalf("wrong fingerprint. got=%s expected=%s", actualFP, expectedFP)
	}
}

func TestSigning(t *testing.T) {
	ni := getUnsignedValidNarInfo()
	err := ni.AddSignature(testPrivKey)
	if err != nil {
		panic(err)
	}

	for _, sig := range ni.Signatures {
		if strings.EqualFold(sig, testSig) {
			return
		}
	}
	t.Fatalf("didn't find valid sig (%s). signatures=%v", testSig, ni.Signatures)
}

/*
func TestSigningReal(t *testing.T) {
	// this one will actually try to load the Nix archive from the local store + then sign

	// note to self: this flushed out problems with (narhash differences, then the sha256: prefix) + (deriver loses /nix/store prefix..?)

	// TODO: should narInfoForPath derive the nar name from the item automatically?
	ni, err := narInfoForPath(testPath, "nar/p9vy4sgsh6m2kgph9i2mv3qgr3iy1afc.tar.xz", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	ni.Signatures = []string{}
	err = ni.AddSignature(testPrivateKey)
	if err != nil {
		t.Fatal(err)
	}

	for _, sig := range ni.Signatures {
		if strings.EqualFold(sig, testPathSig) {
			return
		}
	}
	t.Fatalf("didn't find valid sig. signatures=%v needed=%s", ni.Signatures, testPathSig)
}

func TestConvertToBase32(t *testing.T) {
	expectedHash := "12wsyb5qicqwaa2nz206b9m8y5bz9xa7fx4kizrbx9cjxm828p7s"
	// TODO: where to add "sha256:" prefix that is in the fields in the narinfo???
	h, err := nixToBase32("sha256-+lwkUO2Spb7yj5N0d1RPfxWPaloGiG+FUhyziMvymos=")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.EqualFold(h, expectedHash) {
		t.Fatalf("got wrong hash! got: %v", h)
	}
}
*/
