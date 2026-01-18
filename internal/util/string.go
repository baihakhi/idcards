package util

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

const (
	PathToAssets   string = "static/assets/"
	PathToUploads  string = "uploads/"
	PathToCard     string = "tmp/idcards/"
	PathToContract string = "tmp/contracts/"

	pathToFont string = PathToAssets + "fonts/"
)

func ParseInt(s string) int {
	i, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return i
}

func NormalizeName(nama string) string {
	maxChar := 20
	if len(nama) <= maxChar {
		return nama
	}
	fN := strings.Split(nama, " ")
	if len(fN) == 1 {
		return fN[0][:maxChar]
	}
	if len(fN[0]) >= 7 && fN[0][:1] == "M" && fN[0][len(fN[0])-2:] == "AD" {
		fN[0] = "M"
	}
	if len(strings.Join(fN[0:3], " ")) >= maxChar-2 {
		fN[2] = fN[2][:1]
	}
	res := strings.Join(fN, " ")
	if len(res) < maxChar {
		maxChar = len(res)
	}

	return res[:maxChar]
}

func CompletionCheck(m map[string]string) (bool, string) {
	check := false
	for key, value := range m {
		if value == "" {
			return check, fmt.Sprintf("%s masih kosong!", key)
		} else {
			check = true
		}
	}
	return check, ""
}

func StringtoByte(str string) ([]byte, error) {
	data := str[strings.Index(str, ",")+1:]
	imgByte, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}
	return imgByte, nil
}

// func GetFileFormat(filePath string) string {
// 	filename := strings.Split(filePath, "/")
// 	fileFormat := strings.Split(filename[len(filename)-1], ".")[2]
// 	return fileFormat
// }
// --- IGNORE ---
