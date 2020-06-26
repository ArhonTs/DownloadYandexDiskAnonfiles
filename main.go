package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"

	"github.com/dustin/go-humanize"
	"github.com/opesun/goquery"
)

type WriteCounter struct {
	Total uint64
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += uint64(n)
	wc.PrintProgress()
	return n, nil
}

func (wc WriteCounter) PrintProgress() {
	// Clear the line by using a character return to go back to the start and remove
	// the remaining characters by filling it with spaces
	fmt.Printf("\r%s", strings.Repeat(" ", 35))

	// Return again and print current status of download
	// We use the humanize package to print the bytes in a meaningful way (e.g. 10 MB)
	fmt.Printf("\rDownloading... %s complete", humanize.Bytes(wc.Total))
}

//получаем url скачивания в зависимости от файлообменника
func GetUrl(urls string) (rez string, name string) {
	var re = regexp.MustCompile(`(?m)(https?://.+\.\w+)`)

	for _, match := range re.FindAllString(urls, -1) {
		switch match {
		case "https://anonfiles.com":
			x, _ := goquery.ParseUrl(urls)
			url := x.Find("#download-url").Attrs("href")
			rez = url[0]
			name = path.Base(rez)
		case "https://yadi.sk":
			x, err := http.Get("https://cloud-api.yandex.net/v1/disk/public/resources/download?public_key=" + urls)
			if err != nil {
				log.Fatalln(err)
			}
			var result map[string]interface{}
			json.NewDecoder(x.Body).Decode(&result)
			rez = fmt.Sprintf("%v", result["href"])
			//вырезаем имя файла из строки
			beginstr := strings.LastIndex(rez, "&filename=")
			endstr := strings.LastIndex(rez, "&disposition=")
			//декодируем имя в читаемый вид
			unescapedPath, err := url.PathUnescape(rez[beginstr+10 : endstr])
	        if err != nil {
		    log.Fatal(err)
	        }
			name = unescapedPath


		default:
			rez = urls
			name = path.Base(rez)
		}
	}
	return rez, name
}
func main() {

	const LIMIT = 8
	var throttler = make(chan int, LIMIT)
	var rez, name string

	file, err := readFile("file.txt")
	if err != nil {
		log.Println("Can't read file url.")
		os.Exit(1)
	}
	var wg sync.WaitGroup
	for _, file := range file {
		throttler <- 0
		wg.Add(1)
		rez, name = GetUrl(file)
		fmt.Println(name)
		go DownloadFile(&wg, rez, name)
	}
	//Ожидаем завершение всех потоков
	wg.Wait()
	fmt.Println("Загрузка успешно завершена")
}

///Читаем файл
func readFile(f string) (data []string, err error) {
	b, err := os.Open(f)
	if err != nil {
		return
	}
	defer b.Close()

	scanner := bufio.NewScanner(b)
	for scanner.Scan() {
		data = append(data, scanner.Text())
	}
	return
}

func DownloadFile(wg *sync.WaitGroup, url, names string) error {
	defer wg.Done()
	// Create the file, but give it a tmp file extension, this means we won't overwrite a
	// file until it's downloaded, but we'll remove the tmp extension once downloaded.
	out, err := os.Create(names + ".tmp")
	if err != nil {
		return err
	}
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		out.Close()
		return err
	}
	defer resp.Body.Close()

	// Create our progress reporter and pass it to be used alongside our writer
	counter := &WriteCounter{}
	if _, err = io.Copy(out, io.TeeReader(resp.Body, counter)); err != nil {
		out.Close()
		return err
	}

	// The progress use the same line so print a new line once it's finished downloading
	fmt.Print("\n")

	// Close the file without defer so it can happen before Rename()
	out.Close()

	if err = os.Rename(names+".tmp", names); err != nil {
		return err
	}
	return nil

}
