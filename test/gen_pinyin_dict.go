package main

// https://raw.githubusercontent.com/mozillazg/pinyin-data/master/pinyin.txt

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"
)

type cmdArgs struct {
	inputFile  string
	outputFile string
}

func downloadFile() bool {
	durl := "https://raw.githubusercontent.com/mozillazg/pinyin-data/master/pinyin.txt"
	uri, err := url.ParseRequestURI(durl)
	if err != nil {
		println(fmt.Sprintf("check url error. %s", err.Error()))
		return false
	}
	filename := path.Base(uri.Path)

	client := http.DefaultClient
	client.Timeout = time.Second * 60 //设置超时时间
	resp, err := client.Get(durl)
	if err != nil {
		println(fmt.Sprintf("download file error. %s", err.Error()))
		return false
	}
	raw := resp.Body
	defer raw.Close()
	reader := bufio.NewReader(raw)
	file, err := os.Create(filename)
	if err != nil {
		println(fmt.Sprintf("create file error. %s", err.Error()))
		return false
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	reader.WriteTo(writer)
	return true
}

func genCode(inFile *os.File, outFile *os.File) {
	rd := bufio.NewReader(inFile)
	output := `package gopsu

// PinyinDict is data map
// Warning: Auto-generated file, don't edit.
var pinyinDict = map[int]string{
`
	lines := []string{}

	for {
		line, err := rd.ReadString('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
		if strings.HasPrefix(line, "#") {
			continue
		}

		// line: `U+4E2D: zhōng,zhòng  # 中`
		dataSlice := strings.Split(line, "  #")
		dataSlice = strings.Split(dataSlice[0], ": ")
		// 0x4E2D
		hexCode := strings.Replace(dataSlice[0], "U+", "0x", 1)
		// zhōng,zhòng
		pinyin := dataSlice[1]
		lines = append(lines, fmt.Sprintf("\t%s: \"%s\",", hexCode, pinyin))
	}

	output += strings.Join(lines, "\n")
	output += "\n}\n"
	outFile.WriteString(output)
	return
}

func parseCmdArgs() cmdArgs {
	flag.Parse()
	inputFile := flag.Arg(0)
	outputFile := flag.Arg(1)
	return cmdArgs{inputFile, outputFile}
}

func main() {
	if !downloadFile() {
		// println("download pinyin.txt failed.")
		os.Exit(1)
	}
	// args := parseCmdArgs()
	usage := "gen_pinyin_dict INPUT OUTPUT"
	inputFile := "pinyin.txt"
	outputFile := "../pinyin_dict.go"
	if inputFile == "" || outputFile == "" {
		fmt.Println(usage)
		os.Exit(1)
	}

	inFp, err := os.Open(inputFile)
	if err != nil {
		fmt.Printf("open file %s error", inputFile)
		panic(err)
	}
	outFp, err := os.Create(outputFile)
	if err != nil {
		fmt.Printf("open file %s error", outputFile)
		panic(err)
	}
	defer inFp.Close()
	defer outFp.Close()

	genCode(inFp, outFp)
	println("done.")
}
