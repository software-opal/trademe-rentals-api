package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/net/html"
)

type badSearchPage struct{}

func (p badSearchPage) Error() string {
	return "badSearchPage"
}

type propertyData struct {
	uri                          *url.URL
	title, location, description string
	otherData                    map[string]string
	price                        float32
	images                       []*url.URL
}

var propertyURLRegex = regexp.MustCompile("http[s]?:\\/\\/www\\.trademe\\.co\\.nz\\/property\\/residential-property[a-z\\-]+?/auction-\\d+.htm")

func multiplex(multiplexCount int, execFunc func(chan *propertyData)) chan *propertyData {
	// See: https://blog.golang.org/pipelines#TOC_4.
	outChan := make(chan *propertyData, 2)
	var wg sync.WaitGroup
	wg.Add(multiplexCount)
	fn := func() {
		execFunc(outChan)
		wg.Done()
	}
	for i := 0; i < multiplexCount; i++ {
		go fn()
	}
	go func() {
		wg.Wait()
		close(outChan)
	}()
	return outChan
}

func main() {
	// fmt.Println("Starting")
	urlStrings := []string{}
	urlStrings = append(urlStrings,
		"http://www.trademe.co.nz/browse/categoryattributesearchresults.aspx?cid=5748&search=1&134=1&135=7&59=20000%2c40000&rptpath=350-5748-&nofilters=1&originalsidebar=1&key=966459328&page=1&sort_order=prop_default",
		"http://www.trademe.co.nz/browse/categoryattributesearchresults.aspx?cid=5748&search=1&134=1&135=7&59=20000%2c40000&rptpath=350-5748-&nofilters=1&originalsidebar=1&key=966459328&page=2&sort_order=prop_default",
		"http://www.trademe.co.nz/browse/categoryattributesearchresults.aspx?cid=5748&search=1&134=1&135=7&59=20000%2c40000&rptpath=350-5748-&nofilters=1&originalsidebar=1&key=966459328&page=3&sort_order=prop_default",
	)

	propURLs := make(chan *url.URL, 10)
	go func() {
		for _, urlString := range urlStrings {
			uri, _ := url.Parse(urlString)
			fmt.Println("Reading ", uri)
			err := loadPropertyFromURLList(uri, propURLs)
			if err != nil {
				fmt.Println("Failed to read uri due to error: ", err)
			}
		}
		close(propURLs)
	}()
	propData := multiplex(5, func(outChan chan *propertyData) {
		for propURL := range propURLs {
			fmt.Println(propURL)
			outChan <- loadPropertyDataFrom(propURL)
		}
	})
	for data := range propData {
		fmt.Println(data.price)
	}
}

func loadPropertyFromURLList(baseURL *url.URL, properties chan *url.URL) error {
	resp, err := http.Get(baseURL.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	// buf.ReadFrom(resp.Body)
	fmt.Println(buf.String())
	z := html.NewTokenizer(resp.Body)
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			// Probably the end of the file ...
			fmt.Println("Error: ", z.Err())
			break
		}
		if tt == html.StartTagToken {
			tagName := getTagName(z)
			attrMap := getAttrs(z)
			switch tagName {
			case "a":
				href, ok := attrMap["href"]
				if ok {
					tagURL, err := getPropertyURL(baseURL, href)
					if err == nil && tagURL != nil {
						fmt.Println("Got URL", tagURL)
						properties <- tagURL
					}
				}
			case "div":
				// Handle error page.
				id, ok := attrMap["id"]
				if ok && id == "ErrorOops" {
					return badSearchPage{}
				}
			}
		}
	}
	fmt.Println("Done loading search results")
	return nil
}

func getTagName(z *html.Tokenizer) string {
	nameB, _ := z.TagName()
	return string(nameB)
}
func ifTag(z *html.Tokenizer, tagName string) bool {
	nameB, _ := z.TagName()
	return string(nameB) == tagName
}

func getPropertyURL(baseURL *url.URL, hrefValue string) (*url.URL, error) {
	tagURL, err := url.Parse(hrefValue)
	if err == nil {
		url := baseURL.ResolveReference(tagURL)
		if propertyURLRegex.MatchString(url.String()) {
			return url, nil
		}
		return nil, nil
	}
	return nil, err
}

func getAttrs(z *html.Tokenizer) map[string]string {
	attrMap := make(map[string]string, 10)
	kB, vB, _ := z.TagAttr()
	for len(kB) > 0 {
		k, v := string(kB), string(vB)
		attrMap[k] = v
		kB, vB, _ = z.TagAttr()
	}
	return attrMap
}
func loadPropertyDataFrom(propertyURL *url.URL) *propertyData {
	resp, err := http.Get(propertyURL.String())
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	z := html.NewTokenizer(resp.Body)
	if err := findTagWithAttr(z, "div", "id", "mainContent"); err != nil {
		return loadDataFromMainContent(propertyURL, z)
	}

	return nil
}
func findTagWithAttr(z *html.Tokenizer, tagName, attrName, attrValue string) error {
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			return z.Err()
		} else if tt == html.StartTagToken {
			currTagName := getTagName(z)
			attrMap := getAttrs(z)
			if currTagName == tagName {
				currAttrValue, ok := attrMap[attrName]
				if ok && currAttrValue == attrValue {
					return nil
				}
			}
		}
	}
}
func readText(z *html.Tokenizer) (string, error) {
	text := ""
	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			return "", z.Err()
		case html.TextToken:
			text = text + string(z.Text())
		default:
			return strings.Trim(text, " \n\t"), nil
		}
	}
}
func readTextFromTagWithAttr(z *html.Tokenizer, tagName, attrName, attrValue string) (string, error) {
	err := findTagWithAttr(z, tagName, attrName, attrValue)
	if err != nil {
		return "", err
	}
	text, err := readText(z)
	if err != nil {
		return "", err
	}
	return text, nil
}
func loadDataFromMainContent(uri *url.URL, z *html.Tokenizer) *propertyData {
	data := new(propertyData)
	re := regexp.MustCompile(`\$(\d+(?:\.\d{2})?)`) // Matches $(123(.45)?)
	fmt.Println("Loading ", uri)
	// Find <h1 id="ListingTitle_title">
	text, err := readTextFromTagWithAttr(z, "h1", "id", "ListingTitle_title")
	if err != nil {
		return nil
	}
	data.title = text

	text, err = readTextFromTagWithAttr(z, "li", "id", "ListingTitle_classifiedTitlePrice")
	if err != nil {
		return nil
	}
	matches := re.FindStringSubmatch(text)
	if len(matches) != 2 {
		return nil
	}
	priceF, err := strconv.ParseFloat(matches[1], 32)
	if err != nil {
		return nil
	}
	data.price = float32(priceF)

	return data
}
