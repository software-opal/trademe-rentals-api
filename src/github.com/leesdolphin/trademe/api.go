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
      "http://www.trademe.co.nz/browse/categoryattributesearchresults.aspx?cid=5748&search=1&v=list&nofilters=1&originalsidebar=1&134=1&135=7&59=20000%2c40000&rptpath=350-5748-&key=962308185&page=1&sort_order=prop_default",
      "http://www.trademe.co.nz/browse/categoryattributesearchresults.aspx?cid=5748&search=1&v=list&134=1&135=7&59=20000%2c40000&rptpath=350-5748-&nofilters=1&originalsidebar=1&key=962308185&page=2&sort_order=prop_default",
      "http://www.trademe.co.nz/browse/categoryattributesearchresults.aspx?cid=5748&search=1&v=list&nofilters=1&originalsidebar=1&134=1&135=7&59=20000%2c40000&rptpath=350-5748-&key=962308185&page=3&sort_order=prop_default",
    )

  propUrls := make(chan *url.URL, 10)
  for _, urlString := range urlStrings {
    uri, _ := url.Parse(urlString)
    go LoadPropertyFromUrlList(uri, propUrls)
  }
  propData := make(chan *PropertyData, 100)
  for i:=0;i<5;i++ {
    go func() {
      for {
        data, ok := <-propData
        fmt.Println(data.price)
      }
    }()
  }
}


func LoadPropertyFromUrlList(baseUrl *url.URL, properties chan *url.URL) {
  resp, err := http.Get(baseUrl.String())
  if err != nil {
    return
  }
  defer resp.Body.Close()
  z := html.NewTokenizer(resp.Body)
  for {
    tt := z.Next()
    if tt == html.ErrorToken {
      break
    }
    if tt == html.StartTagToken  {
      name_b, hasAttrs := z.TagName()
      name := string(name_b)
      if name == "a" && hasAttrs {
        for key_b, value_b, hasAttrs := z.TagAttr() ;hasAttrs; key_b, value_b, hasAttrs = z.TagAttr() {
          if string(key_b) == "href" {
            tagUrl, err := getPropertyUrl(baseUrl, value_b)
            if err == nil && tagUrl != nil {
              properties <- tagUrl
            }
          }
        }
      }
    }
  }
}


func getPropertyUrl(baseUrl *url.URL, hrefValue []byte) (*url.URL, error) {
  tagUrl, err := url.Parse(string(hrefValue))
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

func LoadPropertyDataFrom(propertyUrl *url.URL) *PropertyData {
  resp, err := http.Get(baseUrl.String())
  if err != nil {
    return
  }
  defer resp.Body.Close()
  z := html.NewTokenizer(resp.Body)
}
