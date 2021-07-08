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
	"github.com/PuerkitoBio/goquery"
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
	// Clear the line by using a1 character return to go back to the start and remove
	// the remaining characters by filling it with spaces
	fmt.Printf("\r%s", strings.Repeat(" ", 35))

	// Return again and print current status of download
	// We use the humanize package to print the bytes in a meaningful way (e.g. 10 MB)
	fmt.Printf("\rDownloading... %s complete", humanize.Bytes(wc.Total))
}
func ExecUrl(url string) (*goquery.Document, error){
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
    }
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
			log.Fatalln(err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:78.0) Gecko/20100101 Firefox/78.0")
	resp, err := client.Do(req)
	if err != nil {
			log.Fatalln(err)
	}
	defer resp.Body.Close()	
return goquery.NewDocumentFromReader(resp.Body)
}

//получаем url скачивания в зависимости от файлообменника
func GetUrl(urls string) (rez string, name string) {
	var re = regexp.MustCompile(`(?m)(https?://.+\.\w+/)`)
	for _, match := range re.FindAllString(urls, -1) {
		switch match {	
		case "https://anonfiles.com/","https://anonfile.com/":
			x,_:= ExecUrl(urls)
			url,_ := x.Find("#download-url").Attr("href")
			rez = url
			name = path.Base(rez)
		case "https://yadi.sk/":
			x, err := http.Get("https://cloud-api.yandex.net/v1/disk/public/resources/download?public_key=" + urls)
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
		case "https://nofile.org/":
			x,_:= ExecUrl(urls)
			url,_ := x.Find("a.btn").Attr("href")
			rez = "https:"+url
			fmt.Println(rez)
			//urlname := x.Find("ul li:first-child").Text()
			name = "AutoLink_2020_Cs50.zip"
		case "https://www.upload.ee/": 
		x,_:= ExecUrl(urls)
		url,_ := x.Find("#d_l").Attr("href")
		fmt.Println(url)
		rez = url
		name = path.Base(rez)
		case "https://drive.google.com/":
			beginstr := strings.LastIndex(urls, "/d/")
			endstr := strings.LastIndex(urls, "/view")
			//вырезаем id
			idstr := urls[beginstr+3:endstr]
			fmt.Println(idstr)
			//получаем прямую ссылку загрузки
			rez = "https://docs.google.com/uc?authuser=1&id="+idstr+"&export=download"
			fmt.Println(rez)
			name ="12.rar"
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
		go DownloadFile(&wg, rez, name)
	}
	//Ожидаем завершение всех потоков
	wg.Wait()
	fmt.Println("Загрузка всех файлов успешно завершена")
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
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
			log.Fatalln(err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:78.0) Gecko/20100101 Firefox/78.0")
	resp, err := client.Do(req)
	if err != nil {
			log.Fatalln(err)
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

	fmt.Println("Файл "+names+" успешно загружен")
	return nil

}
