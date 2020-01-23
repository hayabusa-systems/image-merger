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
)

func checkImgType(fileType string) bool {
	return fileType == "jpeg" || fileType == "png"
}

func main() {
	// ディレクトリの確認、オリジナルの画像はoriginal、マージする画像はmerge-item、生成した画像はprocessingに格納する。
	if f, err := os.Stat("original"); os.IsNotExist(err) || !f.IsDir() {
		fmt.Println("合成元の画像を格納するディレクトリが存在しませんでした。実行ファイルと同じディレクトリに original ディレクトリを作成してください。")
		os.Exit(-1)
	}
	if f, err := os.Stat("merge-item"); os.IsNotExist(err) || !f.IsDir() {
		fmt.Println("合成する画像を格納するディレクトリが存在しませんでした。実行ファイルと同じディレクトリに merge-item ディレクトリを作成してください。")
		os.Exit(-1)
	}
	// 出力用のディレクトリが存在しない場合作成する。
	if f, err := os.Stat("processing"); os.IsNotExist(err) || !f.IsDir() {
		if err = os.Mkdir("processing", os.ModeDir); err != nil {
			fmt.Printf("出力用ディレクトリを作成するのに失敗しました。: %s\n", err.Error())
			os.Exit(-1)
		}
	}

	mergeImgFiles, _ := ioutil.ReadDir("./merge-item")
	if len(mergeImgFiles) == 0 {
		fmt.Println("合成する画像が存在しませんでした。merge-item ディレクトリに合成したい画像を格納してください。")
		os.Exit(-1)
	}
	if len(mergeImgFiles) > 1 {
		fmt.Println("合成する画像が2つ以上存在します。1つしか合成することができません。")
		os.Exit(-1)
	}

	// image.Decodeのunexpected EOF対策
	imgHeader := bytes.NewBuffer(nil)
	mergeImgSrc, _ := os.Open("merge-item/" + mergeImgFiles[0].Name())
	mergeImgReader := io.TeeReader(mergeImgSrc, imgHeader)

	_, mergeImgType, err := image.DecodeConfig(mergeImgReader)
	if err != nil {
		fmt.Printf("画像読み込み処理でエラーが発生しました。\n\n%s\n", err.Error())
		os.Exit(-1)
	}
	if !checkImgType(mergeImgType) {
		fmt.Println("合成する画像はJPEG またはPNGを指定する必要があります。")
		os.Exit(-1)
	}

	var mergeImg image.Image
	mergeImageMultiReader := io.MultiReader(imgHeader, mergeImgSrc)
	if mergeImgType == "jpeg" {
		mergeImg, err = jpeg.Decode(mergeImageMultiReader)
	} else {
		mergeImg, err = png.Decode(mergeImageMultiReader)
	}
	if err != nil {
		fmt.Printf("画像読み込み処理でエラーが発生しました。\n\n%s\n", err.Error())
		os.Exit(-1)
	}

	originalImgFiles, _ := ioutil.ReadDir("./original")
	if len(originalImgFiles) == 0 {
		fmt.Println("合成元の画像が存在しませんでした。original ディレクトリに合成したい画像を格納してください。")
		os.Exit(-1)
	}
	for _, originalImgFile := range originalImgFiles {
		originalImgSrc, _ := os.Open("original/" + originalImgFile.Name())
		originalImgReader := io.TeeReader(originalImgSrc, imgHeader)

		_, originalImgType, err := image.DecodeConfig(originalImgReader)
		if err != nil {
			fmt.Printf("画像読み込み処理でエラーが発生しました。\n\n%s\n", err.Error())
			os.Exit(-1)
		}
		if !checkImgType(originalImgType) {
			fmt.Println("合成する画像はJPEG またはPNGを指定する必要があります。")
			os.Exit(-1)
		}

		var originalImg image.Image
		originalImageMultiReader := io.MultiReader(imgHeader, originalImgSrc)
		if originalImgType == "jpeg" {
			originalImg, err = jpeg.Decode(originalImageMultiReader)
		} else {
			originalImg, err = png.Decode(originalImageMultiReader)
		}
		if err != nil {
			fmt.Printf("画像読み込み処理でエラーが発生しました。\n\n%s\n", err.Error())
			os.Exit(-1)
		}

		rgba := image.NewRGBA(originalImg.Bounds())
		draw.Draw(rgba, originalImg.Bounds(), originalImg, image.Point{0, 0}, draw.Src)
		draw.Draw(rgba, mergeImg.Bounds(), mergeImg, image.Point{0, 0}, draw.Over)

		// 出力先に既にファイルが存在する場合、一旦消す
		outputPath := "processing/" + originalImgFile.Name()
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
