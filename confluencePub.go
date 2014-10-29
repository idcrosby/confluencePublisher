package main

import (
	"bufio"
	"bytes"
	"flag"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"text/template"
	"time"
)

var InfoLog *log.Logger
var ErrorLog *log.Logger
var Verbose bool


var configuration Configuration

// Templates
var sampleTemplateFile = "resources/sampleHtml.txt"

func main() {

	var logFileName string
	var writer io.Writer

	// Define flags
	flag.BoolVar(&Verbose, "verbose", false, "Turn on verbose logging.")
	flag.StringVar(&logFileName, "logFile", "", "Verbose log to file.")
	flag.Parse()

	// init loggers
	if len(logFileName) > 0 {
		logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			fmt.Printf("Error opening log file: ", err)
		} else {
			defer logFile.Close()
			writer = bufio.NewWriter(logFile)
		}
	} else {
		writer = os.Stdout
	}

	InfoLog = log.New(writer, "INFO: ", log.LstdFlags)
	ErrorLog = log.New(writer, "ERROR: ", log.LstdFlags)

	// Read Config
	file, err := os.Open("conf.json")
	check(err)
	decoder := json.NewDecoder(file)
	configuration = Configuration{}
	err = decoder.Decode(&configuration)
	check(err)
	
	pageData := fetchExternalData()
	
	for _, page := range pageData {
		buildNewConfluencePage(page)
	}
}

func fetchExternalData() []MyPageData {
	// Implement
	var result []MyPageData
	result = make([]MyPageData, 1)
	result[0] = MyPageData{Title: "fake title"}
	return result
}

func buildNewConfluencePage(config MyPageData) {

	// first check if exists
	oldPage := fetchPageByName(config.Title)
	var method string
	pageUrl := configuration.ConfluenceHost + "rest/api/content/"
	page := ConfluencePage{Type: "page", Title: config.Title}

	if len(oldPage.Id) > 0 {
		// Update
		fmt.Println("Update Page")
		method = "PUT"
		pageUrl += oldPage.Id
		page.Id = oldPage.Id
		page.Version.Number = oldPage.Version.Number + 1
	} else {
		method = "POST"
	}

	page.Space.Key = configuration.ConfluenceSpaceKey
	page.Ancestors = make([]Ancestor, 1)
	page.Ancestors[0] = Ancestor{Id: configuration.ConfluenceParentPageId}
	page.Body.Storage.Representation = "storage"
	var pageBuffer bytes.Buffer

	const layout = "Jan 2, 2006 at 3:04pm (MST)"
	pageBuffer.WriteString("<p>Created: " + time.Now().Format(layout) + "</p>")
	sampleTemplate, err := template.ParseFiles(sampleTemplateFile)
	check(err)
	sampleTemplate.Execute(&pageBuffer, config)

	page.Body.Storage.Value = pageBuffer.String()
	
	// Send Confluence API Create
	buff, err := json.Marshal(page)
	check(err)

	req, err := http.NewRequest(method, pageUrl, bytes.NewReader(buff))
	check(err)

	req.SetBasicAuth(configuration.ConfluenceUser, configuration.ConfluencePassword)
	req.Header.Add("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	check(err)

	// Check status
	if res.StatusCode != 200 {
		defer res.Body.Close()
		body, _ := ioutil.ReadAll(res.Body)
		fmt.Println("Error creating confluence page: " + page.Title)
		fmt.Println("Response: ", string(body))
		fmt.Println("Request: ", string(buff))
	}

	fmt.Println("Created confluence page: ", page.Title)
}

func fetchPageByName(pageName string) ConfluencePage {
	url := configuration.ConfluenceHost + "rest/api/content?spaceKey=" + configuration.ConfluenceSpaceKey + "&title=" + url.QueryEscape(pageName) + "&expand=version"
	req, err := http.NewRequest("GET", url, nil)
	check(err)
	req.SetBasicAuth(configuration.ConfluenceUser, configuration.ConfluencePassword)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Error fetching confluence page ", err)
		return ConfluencePage{}
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	var results PageResults
	err = json.Unmarshal(body, &results)
	check(err)
	if len(results.Results) != 1 {
		fmt.Println("Page: " + pageName + " does not exist in space: " + configuration.ConfluenceSpaceKey)
		return ConfluencePage{}
	}
	return results.Results[0]
}

func check(err error) { 
	if err != nil {
		fmt.Println("Panicking: ", err) 
		panic(err) 
	} 
}

type Configuration struct {
	ConfluenceUser string
	ConfluencePassword string
	ConfluenceSpaceKey string
	ConfluenceParentPageId string
	ConfluenceHost string
}


type MyPageData struct {
	Title string
	ListOfData []string
	// Fill
}

// Confluence Model objects

type ConfluencePage struct {
	Id string                        `json:"id,omitempty"`
	Type string                      `json:"type"`
	Title string                     `json:"title"`
	Space struct {
		Key string                   `json:"key"`
		}                            `json:"space"`
	Ancestors []Ancestor             `json:"ancestors"`
	Body struct {
		Storage struct {
			Value string             `json:"value"`
			Representation string    `json:"representation"`
		}                            `json:"storage"`
	}                                `json:"body"`
	Version struct {
		// By struct {
		// 	Type string              `json:"type"`
		// 	Username string          `json:"username"`
		// 	DisplayName string       `json:"displayName"`
		// 	UserKey string           `json:"userKey"`
		// }                            `json:"by"`
		// When string                  `json:"when"`
		// Message string               `json:"message"`
		Number int                   `json:"number"`
		// MinorEdit bool               `json:"minorEdit"`
	}                                `json:"version"`
}

type Ancestor struct {
	Id string                       `json:"id"`
}

type PageResults struct {
	Results []ConfluencePage         `json:"results"`
	Size int                         `json:"size"`
	Start int                        `json:"start"`
}