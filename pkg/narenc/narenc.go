package narenc

import (
	"encoding/binary"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/ulikunitz/xz"
)

// PADLEN is the Nix Padding lenth
const PADLEN int64 = 8

func writePadded(writer io.Writer, bs []byte) error {
	length := int64(len(bs))
	binary.Write(writer, binary.LittleEndian, length)
	_, err := writer.Write(bs)
	if err != nil {
		return err
	}
	padding := make([]byte, (PADLEN-length%PADLEN)%PADLEN)
	_, err = writer.Write(padding)
	if err != nil {
		return err
	}
	return nil
}

func writePaddedReader(writer io.Writer, filename string, length int64) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	binary.Write(writer, binary.LittleEndian, int64(length))
	_, err = io.Copy(writer, f)
	if err != nil {
		return err
	}
	padding := make([]byte, (PADLEN-length%PADLEN)%PADLEN)
	_, err = writer.Write(padding)
	if err != nil {
		return err
	}
	return nil
}

func encodeEntry(writer io.Writer, filename string) error {
	if err := writePadded(writer, []byte("(")); err != nil {
		return err
	}

	if err := writePadded(writer, []byte("type")); err != nil {
		return err
	}

	fi, err := os.Lstat(filename)
	if err != nil {
		return err
	}

	if fi.IsDir() {
		if err = writePadded(writer, []byte("directory")); err != nil {
			return err
		}

		entries, err := ioutil.ReadDir(filename)
		if err != nil {
			return err
		}

		// sort entries

		for _, entry := range entries {
			if err = writePadded(writer, []byte("entry")); err != nil {
				return err
			}
			if err = writePadded(writer, []byte("(")); err != nil {
				return err
			}
			if err = writePadded(writer, []byte("name")); err != nil {
				return err
			}
			if err = writePadded(writer, []byte(entry.Name())); err != nil {
				return err
			}
			if err = writePadded(writer, []byte("node")); err != nil {
				return err
			}
			if err = encodeEntry(writer, filepath.Join(filename, entry.Name())); err != nil {
				return err
			}
			if err = writePadded(writer, []byte(")")); err != nil {
				return err
			}
		}
		// TODO: finish
	} else if fi.Mode().IsRegular() {
		if err = writePadded(writer, []byte("regular")); err != nil {
			return err
		}
		// check mode
		if fi.Mode()&0o111 != 0 {
			if err = writePadded(writer, []byte("executable")); err != nil {
				return err
			}
			if err = writePadded(writer, []byte("")); err != nil {
				return err
			}
		}
		if err = writePadded(writer, []byte("contents")); err != nil {
			return err
		}
		if err = writePaddedReader(writer, filename, fi.Size()); err != nil {
			return err
		}
	} else if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
		if err = writePadded(writer, []byte("symlink")); err != nil {
			return err
		}
		if err = writePadded(writer, []byte("target")); err != nil {
			return err
		}
		target, err := os.Readlink(filename)
		if err != nil {
			return err
		}
		if err = writePadded(writer, []byte(target)); err != nil {
			return err
		}
	} else {
		panic("unknown type")
	}
	if err = writePadded(writer, []byte(")")); err != nil {
		return err
	}
	return nil
}

// Encode creates a NAR stream from pth and writes it to writer
func Encode(writer io.Writer, pth string) error {
	if err := writePadded(writer, []byte("nix-archive-1")); err != nil {
		return err
	}

	if err := encodeEntry(writer, pth); err != nil {
		return err
	}

	return nil
}

// TODO: move this out?
func DumpPathXz(storePath string) (string, error) {
	// compress + upload the NAR
	tempFilePath := filepath.Join(os.TempDir(), "nix-dump-path.tmp") // TODO: add random
	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	xzWriter, err := xz.NewWriter(tempFile)
	if err != nil {
		log.Fatal().Err(err).Msg("xz.NewWriter error")
		return "", err
	}
	defer xzWriter.Close()

	err = Encode(xzWriter, storePath)
	if err != nil {
		log.Fatal().Err(err).Msg("error copying the xz stream to file")
	}

	return tempFilePath, nil
}
