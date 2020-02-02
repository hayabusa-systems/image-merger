package main

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

func checkImgType(fileType string) bool {
	return fileType == "jpeg" || fileType == "png"
}

func main() {
	// 実行ファイルのパスを取得する。
	exe, err := os.Executable()
	if err != nil {
		fmt.Printf("実行ファイルのパスを取得するのに失敗しました。: %s\n", err.Error())
		os.Exit(-1)
	}
	currentDir := filepath.Dir(exe)

	// ディレクトリの確認、オリジナルの画像はoriginal、マージする画像はmerge-item、生成した画像はprocessingに格納する。
	if f, err := os.Stat(filepath.Join(currentDir, "original")); os.IsNotExist(err) || !f.IsDir() {
		fmt.Println("合成元の画像を格納するディレクトリが存在しませんでした。実行ファイルと同じディレクトリに original ディレクトリを作成してください。")
		os.Exit(-1)
	}
	if f, err := os.Stat(filepath.Join(currentDir, "merge-item")); os.IsNotExist(err) || !f.IsDir() {
		fmt.Println("合成する画像を格納するディレクトリが存在しませんでした。実行ファイルと同じディレクトリに merge-item ディレクトリを作成してください。")
		os.Exit(-1)
	}
	// 出力用のディレクトリが存在しない場合作成する。
	if f, err := os.Stat(filepath.Join(currentDir, "processing")); os.IsNotExist(err) || !f.IsDir() {
		if err = os.Mkdir(filepath.Join(currentDir, "processing"), os.ModeDir); err != nil {
			fmt.Printf("出力用ディレクトリを作成するのに失敗しました。: %s\n", err.Error())
			os.Exit(-1)
		}
	}

	mergeImgFiles, _ := ioutil.ReadDir(filepath.Join(currentDir, "merge-item"))
	if len(mergeImgFiles) == 0 {
		fmt.Println("合成する画像が存在しませんでした。merge-item ディレクトリに合成したい画像を格納してください。")
		os.Exit(-1)
	}

	// image.Decodeのunexpected EOF対策

	mergeImgs := []image.Image{}
	for _, mergeImgFile := range mergeImgFiles {
		fmt.Printf("読込中: %s\n", mergeImgFile.Name())
		imgHeader := bytes.NewBuffer(nil)
		mergeImgSrc, _ := os.Open(filepath.Join(currentDir, "merge-item/", mergeImgFile.Name()))
		mergeImgReader := io.TeeReader(mergeImgSrc, imgHeader)

		_, mergeImgType, err := image.DecodeConfig(mergeImgReader)
		if err != nil {
			fmt.Printf("%s は画像ファイルではないためスキップします。\n", mergeImgFile.Name())
			continue
		}
		if !checkImgType(mergeImgType) {
			fmt.Printf("合成する画像はJPEG またはPNGを指定する必要があります。\n%s は画像ファイルではないためスキップします。\n", mergeImgFile.Name())
			continue
		}

		var mergeImg image.Image
		mergeImageMultiReader := io.MultiReader(imgHeader, mergeImgSrc)
		if mergeImgType == "jpeg" {
			mergeImg, err = jpeg.Decode(mergeImageMultiReader)
		} else {
			mergeImg, err = png.Decode(mergeImageMultiReader)
		}
		if err != nil {
			fmt.Printf("画像読み込み処理でエラーが発生しました。\n\nファイル名: %s\n%s\n", mergeImgFile.Name(), err.Error())
			os.Exit(-1)
		}
		mergeImgs = append(mergeImgs, mergeImg)
	}

	originalImgFiles, _ := ioutil.ReadDir(filepath.Join(currentDir, "original"))
	if len(originalImgFiles) == 0 {
		fmt.Println("合成元の画像が存在しませんでした。original ディレクトリに合成したい画像を格納してください。")
		os.Exit(-1)
	}
	for _, originalImgFile := range originalImgFiles {
		fmt.Printf("読込中: %s\n", originalImgFile.Name())
		imgHeader := bytes.NewBuffer(nil)
		originalImgSrc, _ := os.Open(filepath.Join(currentDir, "original", originalImgFile.Name()))
		originalImgReader := io.TeeReader(originalImgSrc, imgHeader)

		_, originalImgType, err := image.DecodeConfig(originalImgReader)
		if err != nil {
			fmt.Printf("%s は画像ファイルではないためスキップします。\n", originalImgFile.Name())
			continue
		}
		if !checkImgType(originalImgType) {
			fmt.Printf("合成する画像はJPEG またはPNGを指定する必要があります。\n%s は画像ファイルではないためスキップします。\n", originalImgFile.Name())
			continue
		}

		var originalImg image.Image
		originalImageMultiReader := io.MultiReader(imgHeader, originalImgSrc)
		if originalImgType == "jpeg" {
			originalImg, err = jpeg.Decode(originalImageMultiReader)
		} else {
			originalImg, err = png.Decode(originalImageMultiReader)
		}
		if err != nil {
			fmt.Printf("画像読み込み処理でエラーが発生しました。\n\nファイル名: %s\n%s\n", originalImgFile.Name(), err.Error())
			os.Exit(-1)
		}

		rgba := image.NewRGBA(originalImg.Bounds())
		draw.Draw(rgba, originalImg.Bounds(), originalImg, image.Point{0, 0}, draw.Src)
		for _, i := range mergeImgs {
			draw.Draw(rgba, i.Bounds(), i, image.Point{0, 0}, draw.Over)
		}

		// 出力先に既にファイルが存在する場合、一旦消す
		outputPath := filepath.Join(currentDir, "processing/", originalImgFile.Name())
		if _, err = os.Stat(outputPath); err == nil {
			if err = os.Remove(outputPath); err != nil {
				fmt.Printf("ファイルの削除に失敗しました。 : %s\n", err.Error())
				os.Exit(-1)
			}
		}
		dst, err := os.Create(outputPath)
		if err != nil {
			fmt.Printf("ファイルの作成に失敗しました。 : %s\n", err.Error())
			os.Exit(-1)
		}
		defer dst.Close()

		if originalImgType == "jpeg" {
			if err = jpeg.Encode(dst, rgba, &jpeg.Options{Quality: 100}); err != nil {
				fmt.Printf("画像の書き込みに失敗しました。 : %s\n", err.Error())
				os.Exit(-1)
			}
		} else {
			if err = png.Encode(dst, rgba); err != nil {
				fmt.Printf("画像の書き込みに失敗しました。 : %s\n", err.Error())
				os.Exit(-1)
			}
		}
	}
}
