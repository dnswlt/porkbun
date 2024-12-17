package main

import (
	"context"
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/dnswlt/porkbun/pkg/api"
	"github.com/dnswlt/porkbun/pkg/porkbun"
)

var (
	printRecords = flag.String("print", "",
		"Comma-separated list of DNS records types (A, AAAA, CNAME, TXT, etc.) to print.\n"+
			"Set to \"all\" to print all records.")

	dyndns = flag.Bool("dyndns", false,
		"If true, runs in dyndns mode:\n"+
			"* obtains the current public IP of the host it is running on\n"+
			"* sets that IP address as the A record of the configured domain")

	ddSubdomain = flag.String("subdomain", "",
		"The subdomain to update in -dyndns mode. Leave empty to update the root domain.")

	ddCheckURL = flag.String("check-url", "",
		"An optional URL that -dyndns mode uses to determine if any DNS update is needed.\n"+
			"If the -check-url is available (a GET request returns any http status code),\n"+
			"then no DNS records will be updated.")

	timeout = flag.Duration("timeout", 60*time.Second,
		"Timeout to use for all Porkbun requests combined.")
)

// ipChanged returns true if there is a typ record ("A", "AAAA") in records that matches name and has ip as its content.
func recordExists(records []*api.Record, typ string, name string, content string) bool {
	for _, r := range records {
		if r.Type != typ {
			continue
		}
		if r.Name == name && r.Content == content {
			return true
		}
	}
	return false
}

func dotjoin(subdom, domain string) string {
	if subdom == "" {
		return domain
	}
	return subdom + "." + domain
}

func doDynDNSUpdate(client *porkbun.Client, records []*api.Record) {
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// Ultra-fast path:
	// If a check URL was specified and is available, assume that the current
	// DNS records are fine (b/c otherwise, we'd expect the check URL to be unreachable).
	if *ddCheckURL != "" {
		client := &http.Client{
			// Don't follow redirects
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		checkCtx, checkCancel := context.WithTimeout(ctx, 5*time.Second)
		defer checkCancel()
		req, err := http.NewRequestWithContext(checkCtx, "GET", *ddCheckURL, nil)
		if err != nil {
			log.Fatalf("Cannot create GET request for %s: %v", *ddCheckURL, err)
		}
		r, err := client.Do(req)
		if err == nil {
			n, _ := io.Copy(io.Discard, r.Body)
			r.Body.Close()
			log.Printf("URL check for %s successful (%s, %d bytes). Skipping DNS update.", *ddCheckURL, r.Status, n)
			return
		}
		log.Printf("URL check for %s failed: %v", *ddCheckURL, err)
	}

	// Get own IP.
	ping, err := client.Ping(ctx)
	if err != nil {
		log.Fatalf("Ping failed: %v", err)
	}
	currentIP := ping.YourIP
	log.Printf("Your IP: %s\n", currentIP)

	// Fast path:
	// If the public DNS record for the domain is identical to our current IP,
	// there is nothing to do.
	domain := dotjoin(*ddSubdomain, client.Config.Domain)
	addrs, err := net.LookupHost(domain)
	if err != nil {
		log.Printf("Failed to look up %q: %v", domain, err)
		log.Fatalf("Please set up an A record before running in -dyndns mode")
	} else {
		for _, addr := range addrs {
			if addr == currentIP {
				log.Printf("Current IP %s matches public DNS record for %q. No update required.", currentIP, domain)
				return
			}
		}
	}

	// If we have requested all records already, check if the right one exists.
	if recordExists(records, "A", domain, currentIP) {
		log.Printf("An A record for %s with IP %s already exists. No update required.",
			domain, currentIP)
		return
	}

	// Update A record for subdoman with current IP.
	ip := net.ParseIP(currentIP)
	if ip == nil || ip.To4() == nil {
		log.Fatalf("Not a valid IPv4 address: %s", currentIP)
	}
	_, err = client.EditAllA(ctx, *ddSubdomain, currentIP)
	if err != nil {
		log.Fatalf("Failed to update A record: %v", err)
	}
	log.Printf("Updated A record for %s to %s", client.Config.Domain, currentIP)
}

func doPrintRecords(client *porkbun.Client) []*api.Record {
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	includeAll := false
	include := make(map[string]bool)
	for _, incl := range strings.Split(*printRecords, ",") {
		if incl == "all" {
			includeAll = true
		} else {
			include[strings.ToUpper(incl)] = true
		}
	}
	// Retrieve and print all DNS records
	recordsResp, err := client.RetrieveAll(ctx)
	if err != nil {
		log.Fatalf("RetrieveAll failed: %v", err)
	}
	records := recordsResp.Records
	var recordLines []string
	for _, r := range records {
		if includeAll || include[r.Type] {
			recordLines = append(recordLines, r.String())
		}
	}
	log.Printf("Your records:\n%s", strings.Join(recordLines, "\n"))
	return records
}

func main() {
	flag.Parse()

	configFile := path.Join(os.Getenv("HOME"), ".porkbungo")
	config, err := porkbun.ReadClientConfig(configFile)
	if err != nil {
		log.Fatalf("Cannot read config: %v", err)
	}
	log.Printf("Read config from %s. Running for domain %q.", configFile, config.Domain)

	client := porkbun.NewClient(config, true)

	var records []*api.Record
	if *printRecords != "" {
		records = doPrintRecords(client)
	}

	if *dyndns {
		doDynDNSUpdate(client, records)
	}
}
