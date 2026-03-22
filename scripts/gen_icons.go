//go:build ignore

package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
)

func main() {
	// Navigation mode icon - gamepad shape (blue)
	navIcon := createIcon(color.RGBA{66, 135, 245, 255}, "nav")
	saveIcon(navIcon, "internal/tray/icons/navigation.png")

	// Launcher mode icon - rocket shape (orange)
	launchIcon := createIcon(color.RGBA{245, 158, 66, 255}, "launch")
	saveIcon(launchIcon, "internal/tray/icons/launcher.png")
}

func createIcon(c color.RGBA, mode string) *image.RGBA {
	size := 22
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	// Fill transparent
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.Set(x, y, color.Transparent)
		}
	}

	if mode == "nav" {
		// Draw a simple gamepad shape
		// Body
		for y := 8; y < 16; y++ {
			for x := 4; x < 18; x++ {
				img.Set(x, y, c)
			}
		}
		// Left grip
		for y := 10; y < 18; y++ {
			for x := 2; x < 6; x++ {
				img.Set(x, y, c)
			}
		}
		// Right grip
		for y := 10; y < 18; y++ {
			for x := 16; x < 20; x++ {
				img.Set(x, y, c)
			}
		}
		// D-pad hint (darker)
		dark := color.RGBA{46, 95, 175, 255}
		for y := 10; y < 14; y++ {
			img.Set(7, y, dark)
		}
		for x := 5; x < 10; x++ {
			img.Set(x, 12, dark)
		}
		// Buttons hint
		img.Set(14, 11, dark)
		img.Set(16, 11, dark)
		img.Set(15, 10, dark)
		img.Set(15, 12, dark)
	} else {
		// Draw a simple rocket shape
		// Body
		for y := 6; y < 18; y++ {
			for x := 9; x < 14; x++ {
				img.Set(x, y, c)
			}
		}
		// Nose
		for y := 3; y < 6; y++ {
			w := 6 - y
			for x := 11 - w/2; x < 11 + w/2 + 1; x++ {
				img.Set(x, y, c)
			}
		}
		// Fins
		dark := color.RGBA{175, 108, 46, 255}
		for y := 14; y < 19; y++ {
			img.Set(7, y, dark)
			img.Set(8, y, dark)
			img.Set(14, y, dark)
			img.Set(15, y, dark)
		}
		// Flame
		flame := color.RGBA{255, 100, 50, 255}
		for y := 18; y < 21; y++ {
			img.Set(10, y, flame)
			img.Set(11, y, flame)
			img.Set(12, y, flame)
		}
	}

	return img
}

func saveIcon(img *image.RGBA, path string) {
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	png.Encode(f, img)
}
