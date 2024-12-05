package main

import (
	"bufio"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/miekg/dns"
)

func resolve(domain string, servers []string, subdomains []string, foundFile os.File, nxdomainFile os.File, errorFile os.File) {
	c := dns.Client{
		Net: "tcp",
	}

	for _, subdomain := range subdomains {
		name := strings.Join([]string{subdomain, domain}, ".")
		m := dns.Msg{}
		m.SetQuestion(dns.Fqdn(name), dns.TypeA)

		found := false

		for i := 0; i < 3; i++ {
			server := servers[rand.Intn(len(servers))]
			// fmt.Println("ask", name, server)

			r, _, err := c.Exchange(&m, server)
			if err != nil {
				// fmt.Println(name, "err", err)
			} else if len(r.Answer) == 0 {
				// fmt.Println(name, "nxdomain")
			} else {
				for _, ans := range r.Answer {
					if _, ok := ans.(*dns.A); ok {
						found = true
						fmt.Println(name)
						foundFile.WriteString(name + "\n")
						break
					}
				}
			}

			if found {
				break
			}

			time.Sleep(time.Duration(rand.Intn(10)) * time.Second)
		}
	}
}

func main() {
	defaultDns := strings.Join([]string{
		"8.8.8.8:53",
		"8.8.4.4:53",
		"1.1.1.1:53",
		"1.0.0.1:53",
		"9.9.9.9:53",
		"149.112.112.112:53",
		"208.67.222.222:53",
		"208.67.220.220:53",
		"8.26.56.26:53",
		"8.20.247.20:53",
	}, ",")

	concurrency := flag.Int("concurrency", 3, "concurrency")
	serversList := flag.String("servers", defaultDns, "comma-separated list of DNS servers to use")
	subdomainList := flag.String("subdomains", "kauppa,store,webshop,shop", "subdomains")
	flag.Parse()

	servers := strings.Split(*serversList, ",")
	subdomains := strings.Split(*subdomainList, ",")

	domainsPath := flag.Arg(0)
	foundPath := flag.Arg(1)
	nxdomainPath := flag.Arg(2)
	errorPath := flag.Arg(3)

	file, err := os.Open(domainsPath)
	if err != nil {
		fmt.Println("Error opening file:", domainsPath, err)
		return
	}
	defer file.Close()

	foundFile, err := os.Create(foundPath)
	if err != nil {
		fmt.Println("Error opening file:", foundPath, err)
		return
	}
	defer foundFile.Close()

	nxdomainFile, err := os.Create(nxdomainPath)
	if err != nil {
		fmt.Println("Error opening file:", nxdomainPath, err)
		return
	}
	defer nxdomainFile.Close()

	errorFile, err := os.Create(errorPath)
	if err != nil {
		fmt.Println("Error opening file:", errorPath, err)
		return
	}
	defer errorFile.Close()

	inflight := make(chan struct{}, *concurrency)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		domain := scanner.Text()
		if domain == "" {
			continue
		}

		inflight <- struct{}{}
		go func(name string, servers []string) {
			defer func() { <-inflight }()
			resolve(domain, servers, subdomains, *foundFile, *nxdomainFile, *errorFile)
		}(domain, servers)

		time.Sleep(time.Millisecond * 500)
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading file:", err)
		os.Exit(1)
	}

	for i := 0; i < cap(inflight); i++ {
		inflight <- struct{}{}
	}
}
