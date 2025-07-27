package util

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

const (
	PathToTempl string = "static/assets/"
	PathToCard  string = "cards/idcards/"

	pathToFont string = "static/assets/fonts/"
)

func GenerateIDCard(templatePath, photoPath, outputPath, name, userID, alamat string) error {
	roboto200 := pathToFont + "Roboto/static/Roboto-Light.ttf"
	roboto400 := pathToFont + "Roboto/static/Roboto-Regular.ttf"

	bgFile, err := os.Open(templatePath)
	if err != nil {
		return err
	}
	defer bgFile.Close()
	bgImg, _, _ := image.Decode(bgFile)

	photoFile, err := os.Open(photoPath)
	if err != nil {
		return err
	}
	defer photoFile.Close()
	photoImg, _, _ := image.Decode(photoFile)

	// TODO:Resize photo

	card := image.NewRGBA(bgImg.Bounds())
	draw.Draw(card, bgImg.Bounds(), bgImg, image.Point{}, draw.Src)

	photoPosition := image.Rect(155, 320, 490, 770)
	draw.Draw(card, photoPosition, photoImg, image.Point{}, draw.Over)

	err = drawText(card, name, 240, 818, 32, roboto400, color.Black)
	if err != nil {
		return err
	}
	err = drawText(card, "SIK-"+userID, 240, 862, 28, roboto400, color.Black)
	if err != nil {
		return err
	}

	arr := strings.SplitN(alamat, ",", 2)
	_ = drawText(card, arr[0], 240, 910, 24, roboto200, color.Black)
	if len(arr) == 2 {
		_ = drawText(card, strings.TrimLeft(arr[1], " "), 240, 945, 24, roboto200, color.Black)
	}

	outFile, _ := os.Create(outputPath)
	defer outFile.Close()
	return png.Encode(outFile, card)
}

func drawText(img *image.RGBA, text string, x, y int, fontSize float64, fontPath string, col color.Color) error {
	// Load TTF font
	fontBytes, err := os.ReadFile(fontPath)
	if err != nil {
		return err
	}

	ft, err := opentype.Parse(fontBytes)
	if err != nil {
		return err
	}

	face, err := opentype.NewFace(ft, &opentype.FaceOptions{
		Size:    fontSize,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return err
	}
	defer face.Close()

	// Set up drawer
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: face,
		Dot:  fixed.P(x, y),
	}

	d.DrawString(text)
	return nil
}
