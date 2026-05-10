// genicon 在仓库根目录生成 Icon.png，供 fyne / fyne-cross 打包使用（无 Python/PIL 依赖）。
package main

import (
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
)

func main() {
	const size = 256
	// 与原先 CI 脚本中 PIL 占位色一致
	c := color.RGBA{R: 73, G: 109, B: 137, A: 255}
	rgba := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			rgba.Set(x, y, c)
		}
	}
	f, err := os.Create("Icon.png")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, rgba); err != nil {
		log.Fatal(err)
	}
}
