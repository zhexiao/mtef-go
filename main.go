package main

import (
	"mtef-go/eqn"
	"mtef-go/docx"
	"fmt"
	"github.com/urfave/cli"
	"os"
	"time"
	"log"
)

func main() {
	var filepath, docxDocument string

	app := cli.NewApp()
	app.Name = "Mtef"
	app.Usage = "Convert MSDocx Mathtype Ole object to Latex code"
	app.Version = "2.0"
	app.EnableBashCompletion = true

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "filepath, f",
			Usage:       "Mathtype Ole object filepath",
			Destination: &filepath,
		},
		cli.StringFlag{
			Name:        "wordDocx, w",
			Usage:       "Office word docx documents",
			Destination: &docxDocument,
		},
	}

	app.Action = func(c *cli.Context) error {
		if filepath != "" {
			if _, err := os.Stat(filepath); os.IsNotExist(err) {
				fmt.Println("File not exist!!!!")
				return nil
			}

			//转换数据
			latex := eqn.Convert(filepath)
			fmt.Println(latex)
			return nil
		}

		if docxDocument != "" {
			if _, err := os.Stat(docxDocument); os.IsNotExist(err) {
				fmt.Println("File not exist!!!!")
				return nil
			}

			dw := docx.DocxWord{
				Filename: docxDocument,
				Target:   fmt.Sprintf("/tmp/%v", time.Now().UnixNano()),
			}

			//转换数据
			err := dw.ParseDocx()
			if err != nil {
				return err
			}
		}

		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

	//转换数据,测试开发使用，需要注释上面的所有代码
	//startEqn := 19
	//endEqn := 19
	//for i := startEqn; i <= endEqn; i++ {
	//	pathName := fmt.Sprintf("E:/workspace/goland/src/mtef-go/assets/oleObject%v.bin", i)
	//	latex := eqn.Convert(pathName)
	//	fmt.Println("num:", i, "latex:", latex)
	//}
}
