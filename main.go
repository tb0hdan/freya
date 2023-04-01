package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"freya/webserver"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/go-retryablehttp"
	log "github.com/sirupsen/logrus"
)

const (
	AnswerLineNum      = 3
	DownloadRetrySleep = 10 * time.Second
	UploadRetrysleep   = 60 * time.Second
	// WebServer config.
	ReadTimeout  = 15 * time.Second
	WriteTimeout = 15 * time.Second
	IdleTimeout  = 30 * time.Second
)

var (
	FreyaKey        = os.Getenv("FREYA") // nolint:gochecknoglobals
	Version         = "unset"            // nolint:gochecknoglobals
	GoVersion       = "unset"            // nolint:gochecknoglobals
	Build           = "unset"            // nolint:gochecknoglobals
	BuildDate       = "unset"            // nolint:gochecknoglobals
	MassDNSChecksum = "unset"            // nolint:gochecknoglobals
	XZChecksum      = "unset"            // nolint:gochecknoglobals
)

type Client struct {
	key    string
	logger *log.Logger
}

func (c *Client) SHA256Sum(fname string) (sum string, err error) {
	f, err := os.Open(fname)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func (c *Client) Download(target, filePath string) error {
	// Create the file
	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	retryClient := retryablehttp.NewClient()
	// DefaultClient uses DefaultTransport which in turn has idle connections and keepalives disabled.
	retryClient.HTTPClient = cleanhttp.DefaultClient()
	retryClient.RetryMax = 3
	retryClient.Logger = c.logger

	req, err := retryablehttp.NewRequest(http.MethodGet, target, nil)
	//
	if err != nil {
		return err
	}
	//
	req.Header.Add("X-Session-Token", c.key)
	//
	resp, err := retryClient.Do(req)
	//
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status) // nolint:goerr113
	}

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) Upload(target, filePath string) error {
	file, err := os.Open(filePath)
	//
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("myFile", filepath.Base(file.Name()))
	//
	if err != nil {
		log.Fatal(err)
	}

	_, _ = io.Copy(part, file)

	writer.Close()
	//
	retryClient := retryablehttp.NewClient()
	// DefaultClient uses DefaultTransport which in turn has idle connections and keepalives disabled.
	retryClient.HTTPClient = cleanhttp.DefaultClient()
	retryClient.RetryMax = 3
	retryClient.Logger = c.logger

	req, err := retryablehttp.NewRequest(http.MethodPost, target, body)
	//
	if err != nil {
		return err
	}
	//
	req.Header.Add("X-Session-Token", c.key)
	req.Header.Add("Content-Type", writer.FormDataContentType())

	if err != nil {
		log.Fatal(err)
	}

	response, err := retryClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", response.Status) // nolint:goerr113
	}

	return nil
}

func (c *Client) Check() {
	binarySum, err := c.SHA256Sum("/massdns")
	if err != nil {
		panic(err)
	}

	if binarySum != MassDNSChecksum {
		log.Fatalf("massdns checksum mismatch: `%s` `%s`", binarySum, MassDNSChecksum)
	}

	binarySum, err = c.SHA256Sum("/usr/bin/xz")
	if err != nil {
		panic(err)
	}

	if binarySum != XZChecksum {
		log.Fatalf("xz checksum mismatch: `%s` `%s`", binarySum, XZChecksum)
	}

	if os.Args[0] != "/freya" {
		log.Fatalf("invoke path is wrong")
	}

	current, err := user.Current()
	if err != nil {
		log.Fatalf("could not get current user")
	}

	if current == nil {
		log.Fatalf("could not get current users: err == nil")
	}

	if current.Name != "root" || current.Gid != "0" || current.Uid != "0" {
		log.Fatalf("will not run as non-root")
	}

	if len(c.key) == 0 {
		log.Fatalf("cannot run without FREYA key")
	}
}

func (c *Client) ProcessOutput() {
	res, err := os.Create("/tmp/results.txt") // nolint:gosec
	if err != nil {
		panic(err)
	}

	w := bufio.NewWriter(res)

	f, err := os.Open("/tmp/output.txt")
	if err != nil {
		panic(err)
	}

	lineNum := 0
	scan := bufio.NewScanner(f)

	for scan.Scan() {
		line := scan.Text()
		if strings.Contains(line, "NXDOMAIN") {
			lineNum = 0
		}

		if lineNum > 0 && lineNum <= AnswerLineNum {
			if strings.Contains(line, "NOERROR") {
				domain := ProcessRecord(line)
				_, err = w.WriteString(domain + "\n")

				if err != nil {
					panic(err)
				}
			}
		}
		lineNum++
	}

	_ = w.Flush()
}

func (c *Client) RunMassDNS() {
	dnsCmd := exec.Command("/massdns", "-q", "-r", "/tmp/resolvers.txt", "/tmp/input.txt", "-w", "/tmp/output.txt", "-o", "J")
	err := dnsCmd.Start()
	//
	if err != nil {
		log.Fatal(err.Error())
	}

	err = dnsCmd.Wait()
	if err != nil {
		log.Fatal(err.Error())
	}
}

func (c *Client) Run() {
	var err error
	err = c.Download("https://api.domainsproject.org/api/vo/resolvers", "/tmp/resolvers.txt")

	if err != nil {
		panic(err)
	}

	for {
		err = c.Download("https://api.domainsproject.org/api/vo/download", "/tmp/input.txt")
		if err != nil {
			log.Error(err)
			time.Sleep(DownloadRetrySleep)

			continue
		}
		//
		c.RunMassDNS()
		c.ProcessOutput()
		//
		err = c.Upload("https://api.domainsproject.org/api/vo/upload", "/tmp/results.txt")
		if err != nil {
			log.Error(err)
			time.Sleep(UploadRetrysleep)
		}
		// clean up
		os.Remove("/tmp/input.txt")
		os.Remove("/tmp/output.txt")
		os.Remove("/tmp/results.txt")
	}
}

func ProcessRecord(line string) string {
	domain := strings.Split(line, " ")[0]
	domain = strings.TrimSuffix(domain, ".")
	domain = strings.ToLower(domain)

	return domain
}

func main() {
	logger := log.New()
	client := &Client{
		key:    FreyaKey,
		logger: logger,
	}

	logger.Println("Starting https://domainsproject.org DNS worker - Freya")
	logger.Printf("Build info: version: %s, go: %s, hash: %s, date: %s\n",
		Version,
		GoVersion, Build,
		BuildDate,
	)
	client.Check()
	logger.Println("Self-checks passed...")
	//
	ws := webserver.New(":80", ReadTimeout, WriteTimeout, IdleTimeout)
	ws.SetBuildInfo(Version, GoVersion, Build, BuildDate)

	go ws.Run()
	client.Run()
	logger.Println("Exit...")
}
