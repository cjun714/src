package main

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/cjun714/glog/log"
)

// var listURL = "http://fishtim.com/dev-api/list?page_size=10&page=260"
var listURL = "http://fishtim.com/dev-api/list?page_size=10&page="
var articleURL = "http://fishtim.com/dev-api/article/"
var torrentURL = "http://sinacloud.net/fishtim/"

const pagesize = 10
const pagemax = 259

const listPath = "z:/list.json"
const articlePath = "z:/articles.json"
const imgDir = "z:/image/"
const torrentDir = "z:/torrent/"

func main() {
	log.I("download list")
	list, e := getAllList()
	if e != nil {
		panic(e)
	}

	log.I("download articles")
	articles, e := getAllArticles(list)
	if e != nil {
		panic(e)
	}

	log.I("download images")
	e = getAllImges(list)
	if e != nil {
		panic(e)
	}

	log.I("download torrents")
	e = getAllTorrents(articles)
	if e != nil {
		panic(e)
	}

	log.I("done")
}

func getAllList() ([]InfoEntry, error) {
	list := make([]InfoEntry, 0, pagemax*pagesize)

	for i := 1; i <= pagemax; i++ {
		pagelist, e := getPageList(i)
		if e != nil {
			log.E("failed when getting page:", i, " error:", e)
			return nil, e
		}
		list = append(list, pagelist...)
	}
	log.I("anime count:", len(list))

	log.I("write list:", listPath)
	e := writeObj(list, listPath)
	if e != nil {
		log.E("write list failed：", e)
		return nil, e
	}

	return list, nil
}

func getAllImges(list []InfoEntry) error {
	if _, e := os.Stat(imgDir); os.IsNotExist(e) {
		log.I("mkdir:", imgDir)
		e = os.MkdirAll(imgDir, os.ModeDir)
		if e != nil {
			return e
		}
	}

	count := len(list)
	for i := 0; i < count; i++ {
		if i%100 == 0 {
			log.I("download images:", i)
		}
		if e := download(list[i].Image, imgDir); e != nil {
			log.E("download image failed:", list[i].Image, ", title:", list[i].Title)
		}
	}

	return nil
}

func getAllTorrents(list []Article) error {
	if _, e := os.Stat("torrent"); os.IsNotExist(e) {
		log.I("mkdir:", torrentDir)
		e = os.MkdirAll(torrentDir, os.ModeDir)
		if e != nil {
			return e
		}
	}

	count := len(list)
	for i := 0; i < count; i++ {
		if i%100 == 0 {
			log.I("download torrent:", i)
		}
		tors := list[i].Torrents
		for _, tor := range tors {
			if e := download(torrentURL+tor.TorrPath, torrentDir); e != nil {
				log.E("download torrent failed:", tor.TorrPath, ", title:", list[i].Title)
			}
		}
	}

	return nil
}

func getAllArticles(list []InfoEntry) ([]Article, error) {
	count := len(list)
	articles := make([]Article, 0, count)
	for i := 0; i < count; i++ {
		if i%100 == 0 {
			log.I("download articles:", i)
		}

		article, e := getArticle(list[i].ID)
		if e != nil {
			log.E("download image failed:", list[i].Image, ", title:", list[i].Title)
			continue
		}

		articles = append(articles, *article)
	}

	log.I("write articles:", articlePath)
	e := writeObj(articles, articlePath)
	if e != nil {
		return nil, e
	}

	return articles, nil
}

func writeObj(obj interface{}, path string) error {
	byts, e := json.Marshal(obj)
	if e != nil {
		return e
	}

	return ioutil.WriteFile(path, byts, 0666)
}

func getPageList(page int) ([]InfoEntry, error) {
	respBytes, e := callAPI(listURL + strconv.Itoa(page))
	if e != nil {
		return nil, e
	}

	var resp Response
	e = json.Unmarshal(respBytes, &resp)
	if e != nil {
		return nil, e
	}

	return resp.Results, e
}

func getArticle(id int) (*Article, error) {
	respBytes, e := callAPI(articleURL + strconv.Itoa(id))
	if e != nil {
		return nil, e
	}
	var article Article
	e = json.Unmarshal(respBytes, &article)
	if e != nil {
		return nil, e
	}

	return &article, nil
}

func callAPI(url string) ([]byte, error) {
	client := http.Client{
		Timeout: 30 * time.Second,
	}
	resp, e := client.Get(url)
	if e != nil {
		return nil, e
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, errors.New("http error:" + resp.Status)
	}

	bs, e := ioutil.ReadAll(resp.Body)
	if e != nil {
		return nil, e
	}

	return bs, nil
}

func download(url string, dir string) error {
	resp, e := http.Get(url)
	if e != nil {
		return e
	}
	defer resp.Body.Close()

	name := filepath.Base(url)
	path := filepath.Join(dir, name)
	byts, e := io.ReadAll(resp.Body)
	if e != nil {
		return e
	}
	return ioutil.WriteFile(path, byts, 0666)
}

type Response struct {
	Count    int         `json:"count"`
	Next     string      `json:"next"`
	Previous string      `json:"previous"`
	Results  []InfoEntry `json:"results"`
}

type InfoEntry struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Image       string `json:"image"`
	Info        []Info `json:"year"`
}

type Info struct {
	Year      int `json:"year"`
	ArticleID int `json:"article_id"`
}

type Article struct {
	ID           int       `json:"id"`
	Info         []Info    `json:"year"`
	Torrents     []Torrent `json:"torrent"`
	Title        string    `json:"title"`
	Descriptiion string    `json:"descriptiion"`
	Content      string    `json:"content"`
	Image        string    `json:"image"`
}

type Torrent struct {
	Torr      string `json:"torrent"`      // "MOX7MI6"
	TorrPath  string `json:"torrent_path"` // "bt/MOX7MI6.rar"
	ArticleID int    `json:"article_id"`
}
