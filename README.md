# Tor Scraper (CTI Assignment)

This project is a Tor-based automated onion scraper developed using Go (Golang) for educational Cyber Threat Intelligence (CTI) purposes.

## Project Purpose
The goal of this project is to automatically collect data from Tor-based services by routing all traffic through the Tor network. Manual analysis of large numbers of .onion addresses is impractical, therefore automation is required.

## Features
- Reads target URLs from a file (`targets.yaml`)
- Routes all traffic through Tor SOCKS5 proxy
- Performs connection health checks
- Handles unreachable or dead onion services gracefully
- Generates unique output files per target
- Logs scan results without interrupting execution

## Technologies Used
- Go (Golang)
- net/http
- golang.org/x/net/proxy
- Tor Service (SOCKS5 Proxy)

## Usage
Make sure Tor Browser or a Tor service is running locally.

```bash
go run main.go targets.yaml
