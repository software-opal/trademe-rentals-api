package main

import (
  "fmt"
  "regexp"
  "net/http"
  "net/url"
  "golang.org/x/net/html"
)


var PROPERTY_URL_REGEX *regexp.Regexp = regexp.MustCompile("http[s]?:\\/\\/www\\.trademe\\.co\\.nz\\/property\\/residential-property[a-z\\-]+?/auction-\\d+.htm")

func main() {
  // fmt.Println("Starting")
  urlStrings := []string{}
  urlStrings = append(urlStrings,
      "http://www.trademe.co.nz/browse/categoryattributesearchresults.aspx?cid=5748&search=1&134=1&135=7&59=20000%2c40000&rptpath=350-5748-&nofilters=1&originalsidebar=1&key=964053301&page=1&sort_order=prop_default",
      "http://www.trademe.co.nz/browse/categoryattributesearchresults.aspx?cid=5748&search=1&134=1&135=7&59=20000%2c40000&rptpath=350-5748-&nofilters=1&originalsidebar=1&key=964053301&page=2&sort_order=prop_default",
      "http://www.trademe.co.nz/browse/categoryattributesearchresults.aspx?cid=5748&search=1&134=1&135=7&59=20000%2c40000&rptpath=350-5748-&nofilters=1&originalsidebar=1&key=964053301&page=3&sort_order=prop_default",
    )

  propUrls := make(chan *url.URL, 10)
  go func() {
    for _, urlString := range urlStrings {
      uri, _ := url.Parse(urlString)
      fmt.Println("Reading ", uri)
      LoadPropertyFromUrlList(uri, propUrls)
    }
    close(propUrls)
  }()
  propData := make(chan *PropertyData, 100)
  go func() {
    for propUrl := range propUrls {
      fmt.Println(propUrl)
      propData <- LoadPropertyDataFrom(propUrl)
    }
  }()
  for {
    data := <-propData
    fmt.Println(data.price)
  }
}


func LoadPropertyFromUrlList(baseUrl *url.URL, properties chan *url.URL) {
  fmt.Println("LoadPropertyFromUrlList")
  resp, err := http.Get(baseUrl.String())
  if err != nil {
    return
  }
  defer resp.Body.Close()
  z := html.NewTokenizer(resp.Body)
  fmt.Println("Starting Tokenizer")
  for {
    tt := z.Next()
    if tt == html.ErrorToken {
      break
    }
    if tt == html.StartTagToken  {
      attrMap := ifTagGetAttrs(z, "a")
      if attrMap != nil {
        fmt.Println("Got `a`, has attrs: ", attrMap)
        href, ok := attrMap["href"]
        if ok {
          tagUrl, err := getPropertyUrl(baseUrl, href)
          if err == nil && tagUrl != nil {
            fmt.Println("Got URL", tagUrl)
            properties <- tagUrl
          }
        }
      }
    }
  }
}


func getPropertyUrl(baseUrl *url.URL, hrefValue string) (*url.URL, error) {
  tagUrl, err := url.Parse(hrefValue)
  if err == nil {
    url := baseUrl.ResolveReference(tagUrl)
    if PROPERTY_URL_REGEX.MatchString(url.String()) {
      return url, nil
    } else {
      return nil, nil
    }
  } else {
    return nil, err
  }
}
type PropertyData struct {
  uri *url.URL
  title, location, description string
  otherData map[string]string
  price float32
  images []*url.URL
}
func ifTagGetAttrs(z *html.Tokenizer, tagName string) map[string]string {
  name_b, hasAttrs := z.TagName()
  if string(name_b) != tagName || !hasAttrs {
    return nil
  }
  attrMap := make(map[string]string, 10)
  for k_b, v_b, hasAttrs := z.TagAttr(); hasAttrs; k_b, v_b, hasAttrs = z.TagAttr() {
    k, v := string(k_b), string(v_b)
    attrMap[k] = v
  }
  return attrMap
}
func LoadPropertyDataFrom(propertyUrl *url.URL) *PropertyData {
  resp, err := http.Get(propertyUrl.String())
  if err != nil {
    return nil
  }
  defer resp.Body.Close()
  z := html.NewTokenizer(resp.Body)
  for {
    tt := z.Next()
    if tt == html.ErrorToken {
      return nil
    } else if tt == html.StartTagToken {
      attrMap := ifTagGetAttrs(z, "div")
      if attrMap != nil {
        id, ok := attrMap["id"]
        if ok && id == "mainContent" {
          return loadDataFromMainContent(propertyUrl, z)
        }
      }
    }
  }
  return nil
}
func findTagWithAttr(z *html.Tokenizer, tagName, attrName, attrValue string) bool {
  for {
    tt := z.Next()
    if tt == html.ErrorToken {
      return false
    } else if tt == html.StartTagToken {
      name_b, hasAttrs := z.TagName()
      if string(name_b) == tagName && hasAttrs {
        for k_b, v_b, hasAttrs := z.TagAttr(); hasAttrs; k_b, v_b, hasAttrs = z.TagAttr() {
          if attrName == string(k_b) && attrValue == string(v_b) {
            return true
          }
        }
      }
    }
  }
  return false
}
func loadDataFromMainContent(uri *url.URL, z *html.Tokenizer) *PropertyData {
  fmt.Println("Loading ", uri)
  // Find <h1 id="ListingTitle_title">
  if !findTagWithAttr(z, "h1", "id", "ListingTitle_title") {
    return nil
  }



  return nil
}
