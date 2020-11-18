package main

import (
	"strings"
	"testing"
)

const testPath string = "/nix/store/p9vy4sgsh6m2kgph9i2mv3qgr3iy1afc-firefox-82.0.2"
const testPathSig string = "cache.niche.li:I1ouPYKNyE88Ox0BpelSBdjc/iTi5qKpRFwGQsH4GRL3Q2AMcI+pFkKCskqtNX/Q+UMc+KcVizDigzDYO+86CQ=="
const testPrivateKey string = "cache.niche.li:bMMTjTggxXxCIDZu6zDQq5H67iklXDppZsvSzwq5EyKpW7an8Ohms/zGu2eufEQW8h/eEErCjj5X9+EqQjtQmA=="

type NixMockClient struct{}

func (_ NixMockClient) PathInfo() {}
func (_ NixMockClient) HashFile() {}
func (_ NixMockClient) DumpPath() {
	// pass the known (test) paths over the socket to emulate the build?
}
func (_ NixMockClient) ToBase32()   {}
func (_ NixMockClient) QueryPaths() {}
func (_ NixMockClient) Build()      {}

func getUnsignedValidNarInfo() narInfo {
	/*
		StorePath: /nix/store/p9vy4sgsh6m2kgph9i2mv3qgr3iy1afc-firefox-82.0.2
		URL: nar/1qripqw8z1a63jzkmkgfz2id93qzbjzvq082mny97nah2qr7m4mr.nar.xz
		Compression: xz
		FileHash: sha256:1qripqw8z1a63jzkmkgfz2id93qzbjzvq082mny97nah2qr7m4mr
		FileSize: 1480
		NarHash: sha256:12wsyb5qicqwaa2nz206b9m8y5bz9xa7fx4kizrbx9cjxm828p7s
		NarSize: 8216
		References: 13vlwns812b5hz2zxz3ra1n6g9kiz8bc-gsettings-desktop-schemas-3.38.0 47lx978m7l4k8jraw9pl4dxskv9zc0n4-libkrb5-1.18 4szmklairdmp3qxas7f3sn29fk8216wj-libglvnd-1.3.2 8zd3z4az6y3caga8aw3ip2cp2iv82vjq-systemd-246.6 arndavwmmrlnnk3vqy1wknivi0y95as6-firefox-unwrapped-82.0.2 bjc1jxy0lsy493qdj6dvxi9xa8f2ij99-alsa-lib-1.2.3 d991zcbwlp1s16hpb5qkza1ld7hsnfw1-adwaita-icon-theme-3.38.0 jic97idx4nsq5fd4vghb7sbjwsb4y8nc-libcanberra-0.30 kq664b6brczw415bxfr5sjw8gy7nv5r7-libpulseaudio-13.0 p9vy4sgsh6m2kgph9i2mv3qgr3iy1afc-firefox-82.0.2 pmrhk324fkidrm5ffd5jckb21s9zys6r-bash-4.4-p23 qg7r4zynz1n0lgrn3q3g5nyf5dxjgb0r-ffmpeg-4.3.1 r9lzagx1wj26wai3bwxykf8s0gvnrn5v-gtk+3-3.24.23 v0ljfhxrc7s4apykysqyivzvi4603jfa-libva-2.9.1 zy97dmp14wwgrx6f7wjljlpdn6lb13qk-mesa-20.2.1
		Deriver: nm1q9fwp96zwh2mbbiyn2gh4igk4hmmn-firefox-82.0.2.drv
		Sig: cache.nixos.org-1:4Yqrj5GNEf/CygnSmL6yd+pc4gzHrlL7mOM74ZEwpZqUSFv0C6suuL6Jn+F/qEPsWq6LIUUp2cAgJAUHKyegDQ==
	*/
	return narInfo{
		StorePath: "/nix/store/p9vy4sgsh6m2kgph9i2mv3qgr3iy1afc-firefox-82.0.2",
		URL:       "nar/1qripqw8z1a63jzkmkgfz2id93qzbjzvq082mny97nah2qr7m4mr.nar.xz",
		FileHash:  "",
		FileSize:  1480,
		//NarHash:   "sha256-+lwkUO2Spb7yj5N0d1RPfxWPaloGiG+FUhyziMvymos=", // from nix path-info
		NarHash: "sha256:12wsyb5qicqwaa2nz206b9m8y5bz9xa7fx4kizrbx9cjxm828p7s", // from the narinfo on cachix.org
		NarSize: 8216,
		References: []string{
			"/nix/store/13vlwns812b5hz2zxz3ra1n6g9kiz8bc-gsettings-desktop-schemas-3.38.0",
			"/nix/store/47lx978m7l4k8jraw9pl4dxskv9zc0n4-libkrb5-1.18",
			"/nix/store/4szmklairdmp3qxas7f3sn29fk8216wj-libglvnd-1.3.2",
			"/nix/store/8zd3z4az6y3caga8aw3ip2cp2iv82vjq-systemd-246.6",
			"/nix/store/arndavwmmrlnnk3vqy1wknivi0y95as6-firefox-unwrapped-82.0.2",
			"/nix/store/bjc1jxy0lsy493qdj6dvxi9xa8f2ij99-alsa-lib-1.2.3",
			"/nix/store/d991zcbwlp1s16hpb5qkza1ld7hsnfw1-adwaita-icon-theme-3.38.0",
			"/nix/store/jic97idx4nsq5fd4vghb7sbjwsb4y8nc-libcanberra-0.30",
			"/nix/store/kq664b6brczw415bxfr5sjw8gy7nv5r7-libpulseaudio-13.0",
			"/nix/store/p9vy4sgsh6m2kgph9i2mv3qgr3iy1afc-firefox-82.0.2",
			"/nix/store/pmrhk324fkidrm5ffd5jckb21s9zys6r-bash-4.4-p23",
			"/nix/store/qg7r4zynz1n0lgrn3q3g5nyf5dxjgb0r-ffmpeg-4.3.1",
			"/nix/store/r9lzagx1wj26wai3bwxykf8s0gvnrn5v-gtk+3-3.24.23",
			"/nix/store/v0ljfhxrc7s4apykysqyivzvi4603jfa-libva-2.9.1",
			"/nix/store/zy97dmp14wwgrx6f7wjljlpdn6lb13qk-mesa-20.2.1",
		},
		Deriver:    "nm1q9fwp96zwh2mbbiyn2gh4igk4hmmn-firefox-82.0.2.drv",
		Signatures: []string{},
	}
}

// priv: cache.niche.li:bMMTjTggxXxCIDZu6zDQq5H67iklXDppZsvSzwq5EyKpW7an8Ohms/zGu2eufEQW8h/eEErCjj5X9+EqQjtQmA==
// pub: cache.niche.li:qVu2p/DoZrP8xrtnrnxEFvIf3hBKwo4+V/fhKkI7UJg=
// signature(/nix/store/p9vy4sgsh6m2kgph9i2mv3qgr3iy1afc-firefox-82.0.2)
// "cache.niche.li:I1ouPYKNyE88Ox0BpelSBdjc/iTi5qKpRFwGQsH4GRL3Q2AMcI+pFkKCskqtNX/Q+UMc+KcVizDigzDYO+86CQ==",

func TestSigning(t *testing.T) {
	ni := getUnsignedValidNarInfo()
	err := ni.AddSignature(testPrivateKey)
	if err != nil {
		panic(err)
	}

	for _, sig := range ni.Signatures {
		if strings.EqualFold(sig, testPathSig) {
			return
		}
	}
	t.Fatalf("didn't find valid sig. signatures=%v", ni.Signatures)
}

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
