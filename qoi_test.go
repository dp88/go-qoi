package qoi

import (
	"image"
	_ "image/png"
	"os"
	"testing"
)

const testDir = "./testImages"

func getTestFiles() []string {
	files, err := os.ReadDir(testDir)
	if err != nil {
		panic("Failed to read directory: " + err.Error())
	}

	filePairs := []string{}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// get the file name without the extension
		fileName := file.Name()[:len(file.Name())-4]

		// check if the filePairs slice has the fileName
		shouldAdd := true
		for _, pair := range filePairs {
			if pair == fileName {
				shouldAdd = false
			}
		}

		if shouldAdd {
			filePairs = append(filePairs, fileName)
		}
	}

	return filePairs
}

func TestDecodeConfig(t *testing.T) {
	for _, file := range getTestFiles() {
		fPNG, err := os.Open(testDir + "/" + file + ".png")
		if err != nil {
			t.Fatalf("failed to open file %s: %v", file, err)
		}
		defer fPNG.Close()

		cfgPNG, _, err := image.DecodeConfig(fPNG)
		if err != nil {
			t.Fatalf("failed to decode config for %s.png: %v", file, err)
		}

		fQOI, err := os.Open(testDir + "/" + file + ".qoi")
		if err != nil {
			t.Fatalf("failed to open file %s: %v", file, err)
		}
		defer fQOI.Close()

		cfgQOI, _, err := image.DecodeConfig(fQOI)
		if err != nil {
			t.Fatalf("failed to decode config for %s.qoi: %v", file, err)
		}

		if cfgPNG.Width != cfgQOI.Width || cfgPNG.Height != cfgQOI.Height {
			t.Errorf("mismatched dimensions for %s: PNG(%dx%d) QOI(%dx%d)", file, cfgPNG.Width, cfgPNG.Height, cfgQOI.Width, cfgQOI.Height)
		} else {
			t.Log("dimensions match for", file)
		}
	}
}

func TestDecode(t *testing.T) {
	for _, file := range getTestFiles() {
		fPNG, err := os.Open(testDir + "/" + file + ".png")
		if err != nil {
			t.Fatalf("failed to open file %s: %v", file, err)
		}
		defer fPNG.Close()

		imgPNG, _, err := image.Decode(fPNG)
		if err != nil {
			t.Fatalf("failed to decode image for %s.png: %v", file, err)
		}

		fQOI, err := os.Open(testDir + "/" + file + ".qoi")
		if err != nil {
			t.Fatalf("failed to open file %s: %v", file, err)
		}
		defer fQOI.Close()

		imgQOI, _, err := image.Decode(fQOI)
		if err != nil {
			t.Fatalf("failed to decode image for %s.qoi: %v", file, err)
		}

		for y := 0; y < imgPNG.Bounds().Dy(); y++ {
			for x := 0; x < imgPNG.Bounds().Dx(); x++ {
				r1, g1, b1, a1 := imgPNG.At(x, y).RGBA()
				r2, g2, b2, a2 := imgQOI.At(x, y).RGBA()

				if r1 != r2 || g1 != g2 || b1 != b2 || a1 != a2 {
					t.Errorf("mismatched pixel at (%d, %d) for %s: PNG(%d, %d, %d, %d) QOI(%d, %d, %d, %d)", x, y, file, r1, g1, b1, a1, r2, g2, b2, a2)
				}
			}
		}

		t.Log("pixels match for", file)
	}
}
