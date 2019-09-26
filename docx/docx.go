package docx

import (
	"archive/zip"
	"os"
	"path/filepath"
	"io"
	"io/ioutil"
	"mtef-go/eqn"
	"fmt"
)

type DocxWord struct {
	Filename string
	Target   string
}

//转换文档
func (d *DocxWord) ParseDocx() error {
	err := d.unzip()
	if err != nil{
		return err
	}

	err = d.getLatex()
	if err != nil{
		return err
	}

	return nil
}

//解压缩文件
func (d *DocxWord) unzip() error {
	reader, err := zip.OpenReader(d.Filename)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(d.Target, 0755); err != nil {
		return err
	}

	for _, file := range reader.File {
		path := filepath.Join(d.Target, file.Name)
		if file.FileInfo().IsDir() {
			os.MkdirAll(path, file.Mode())
			continue
		}

		dir := filepath.Dir(path)
		if len(dir) > 0 {
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				err = os.MkdirAll(dir, 0755)
				if err != nil {
					return err
				}
			}
		}

		fileReader, err := file.Open()
		defer fileReader.Close()
		if err != nil {
			return err
		}

		targetFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		defer targetFile.Close()
		if err != nil {
			return err
		}

		if _, err := io.Copy(targetFile, fileReader); err != nil {
			return err
		}
	}

	return nil
}

//转latex
func (d *DocxWord) getLatex() error {
	latexDir := filepath.Join(d.Target, "word/embeddings")
	if _, err := os.Stat(latexDir); os.IsNotExist(err) {
		return err
	}

	dirList, err := ioutil.ReadDir(latexDir)
	if err != nil {
		return nil
	}

	for _, file := range dirList {
		latexFile := filepath.Join(latexDir, file.Name())
		latex := eqn.Convert(latexFile)
		fmt.Println(latex)
	}

	return nil
}
