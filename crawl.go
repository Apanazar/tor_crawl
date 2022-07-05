package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/proxy"
)

var (
	depth     = flag.Int("depth", 1, "select crawler depth")
	target    = flag.String("target", "empty", "specify target link")
	stdout    = flag.String("output", "output.txt", "select the file to output")
	blacklist = flag.String("blacklist", "domains.txt", "select a file to block domains")
	port      = flag.Int("port", 9150, "select the port used by the Tor browser")
)

const (
	ColorRed    = "\u001b[31m"
	ColorGreen  = "\u001b[32m"
	ColorYellow = "\u001b[33m"
	ColorReset  = "\u001b[0m"
)

func repeatCheck(str string) bool {
	data, _ := ioutil.ReadFile(*stdout)

	if !strings.Contains(string(data), str) {
		return true
	} else {
		return false
	}
}

func writeFile(str string) {
	file, err := os.OpenFile(*stdout, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		log.Println(ColorRed, err, ColorReset)
	}
	defer file.Close()

	file.WriteString(str)
}

func getDomains() []string {
	file, err := os.OpenFile(*blacklist, os.O_RDONLY|os.O_WRONLY|os.O_CREATE, 0777)
	scanner := bufio.NewScanner(file)

	if err != nil {
		log.Println(err)
	}
	defer file.Close()

	domain := []string{}
	for scanner.Scan() {
		domain = append(domain, scanner.Text())
	}

	return domain
}

func main() {
	flag.Usage = func() {
		w := flag.CommandLine.Output()
		fmt.Fprintln(w, "Usage of crawler:")
		flag.PrintDefaults()
	}
	flag.Parse()

	collector := colly.NewCollector(
		colly.IgnoreRobotsTxt(),
		colly.MaxDepth(*depth),
		colly.Async(true),
		colly.DisallowedDomains(getDomains()...),
	)

	rp, err := proxy.RoundRobinProxySwitcher(
		fmt.Sprintf("socks5://127.0.0.1:%d", *port),
	)

	if err != nil {
		log.Fatal(err)
	}
	collector.SetProxyFunc(rp)

	collector.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")

		if strings.Contains(link, ".onion") {
			if repeatCheck(link) {
				collector.Visit(e.Request.AbsoluteURL(link))
			}
		}
	})

	collector.OnResponse(func(r *colly.Response) {
		if r.StatusCode == 200 {
			writeFile(fmt.Sprintf("Entrance: %s\n", r.Request.URL.String()))
		}
	})

	collector.OnRequest(func(r *colly.Request) {
		writeFile(fmt.Sprintf("\tLink found:  %s\n", r.URL.String()))
	})

	collector.OnError(func(r *colly.Response, err error) {
		if err != nil {
			if ret := r.Request.Do(); ret != nil {
				return
			}
		}
	})

	fmt.Printf("%sURL: %s\t\t%s%s\n", ColorYellow, ColorGreen, *target, ColorReset)
	fmt.Printf("%sDepth: %s\t\t%d%s\n", ColorYellow, ColorGreen, *depth, ColorReset)
	fmt.Printf("%sOutput: %s\t%s%s\n", ColorYellow, ColorGreen, *stdout, ColorReset)
	fmt.Printf("%sBlacklist: %s\t%s%s\n", ColorYellow, ColorGreen, *blacklist, ColorReset)

	collector.Visit(*target)
	collector.Wait()
}
