package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/go-retryablehttp"
	log "github.com/sirupsen/logrus"
)

var (
	FreyaKey        = os.Getenv("FREYA") // nolint:gochecknoglobals
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

	if len(c.key) == 0 {
		log.Fatalf("cannot run without FREYA key")
	}
}

func (c *Client) RunMassDNS() {
	dnsCmd := exec.Command("/massdns", "-q", "-r", "/tmp/resolvers.txt", "/tmp/input.txt", "-w", "/tmp/output.txt")
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
	err = c.Download("https://api.domainsproject.org/api/vo/download", "/tmp/input.txt")
	fmt.Println(err)
	err = c.Download("https://api.domainsproject.org/api/vo/resolvers", "/tmp/resolvers.txt")
	fmt.Println(err)
	c.RunMassDNS()
	err = c.Upload("https://api.domainsproject.org/api/vo/upload", "/tmp/output.txt")
	fmt.Println(err)
}

func main() {
	logger := log.New()
	client := &Client{
		key:    FreyaKey,
		logger: logger,
	}

	logger.Println("Starting Freya...")
	client.Check()
	logger.Println("Self-checks passed...")
	client.Run()
	logger.Println("Exit...")
}
