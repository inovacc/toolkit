package downloader

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/net/html"
)

var downloadExts = []string{".pdf", ".zip", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".rar", ".tar.gz", ".7z"}

type Downloader struct {
	rules   map[string]bool
	visited map[string]bool
	base    string
}

func NewDownloader() *Downloader {
	return &Downloader{
		rules:   make(map[string]bool),
		visited: make(map[string]bool),
	}
}

func (d *Downloader) Include(v any) bool {
	link, ok := v.(string)
	if !ok {
		return false
	}
	if _, exists := d.rules[link]; exists {
		return false
	}
	d.rules[link] = true
	return true
}

func (d *Downloader) Exclude(v any) bool {
	link, ok := v.(string)
	if !ok {
		return false
	}
	if _, exists := d.rules[link]; !exists {
		return false
	}
	delete(d.rules, link)
	return true
}

func (d *Downloader) List() []string {
	var list []string
	for k := range d.rules {
		list = append(list, k)
	}
	return list
}

func (d *Downloader) Download(outDir string) {
	if len(d.rules) == 0 {
		fmt.Println("No hay archivos para descargar.")
		return
	}

	for link := range d.rules {
		fmt.Println("⬇️  Descargando:", link)
		filename := d.extractFileName(link)
		if filename == "" {
			fmt.Println("❌ No se pudo obtener el nombre del archivo:", link)
			continue
		}

		// Crear archivo en el directorio de salida
		outPath := filepath.Join(outDir, filename)
		err := d.downloadFile(link, outPath)
		if err != nil {
			fmt.Println("❌ Error al descargar:", link, "-", err)
			continue
		}
		fmt.Println("✅ Guardado en:", outPath)
	}
}

func (d *Downloader) extractFileName(link string) string {
	u, err := url.Parse(link)
	if err != nil {
		return ""
	}
	segments := strings.Split(u.Path, "/")
	if len(segments) == 0 {
		return ""
	}
	return segments[len(segments)-1]
}

func (d *Downloader) downloadFile(link, path string) error {
	resp, err := http.Get(link)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status code %d", resp.StatusCode)
	}

	// Crear archivo de destino
	outFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// Copiar contenido al archivo
	_, err = io.Copy(outFile, resp.Body)
	return err
}

func (d *Downloader) Crawler() {
	if d.base == "" {
		fmt.Println("No base URL set. Use Include(baseURL) before calling Crawler.")
		return
	}
	d.crawl(d.base)
}

func (d *Downloader) crawl(current string) {
	if d.visited[current] {
		return
	}
	d.visited[current] = true

	resp, err := http.Get(current)
	if err != nil {
		fmt.Println("Error al acceder:", current, "-", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK || !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/html") {
		return
	}

	z := html.NewTokenizer(resp.Body)
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			break
		}
		if tt == html.StartTagToken || tt == html.SelfClosingTagToken {
			t := z.Token()
			if t.Data == "a" {
				for _, a := range t.Attr {
					if a.Key == "href" {
						link := strings.TrimSpace(a.Val)
						if link == "" || strings.HasPrefix(link, "#") {
							continue
						}
						absLink := d.resolveURL(current, link)
						if absLink == "" {
							continue
						}
						if d.isDownloadable(absLink) {
							if d.Include(absLink) {
								fmt.Println("Descargable encontrado:", absLink)
							}
						} else if d.isSameDomain(d.base, absLink) {
							d.crawl(absLink)
						}
					}
				}
			}
		}
	}
}

func (d *Downloader) resolveURL(base, href string) string {
	baseURL, err1 := url.Parse(base)
	hrefURL, err2 := url.Parse(href)
	if err1 != nil || err2 != nil {
		return ""
	}
	return baseURL.ResolveReference(hrefURL).String()
}

func (d *Downloader) isDownloadable(link string) bool {
	for _, ext := range downloadExts {
		if strings.HasSuffix(strings.ToLower(link), ext) {
			return true
		}
	}
	return false
}

func (d *Downloader) isSameDomain(base, link string) bool {
	baseURL, err1 := url.Parse(base)
	linkURL, err2 := url.Parse(link)
	if err1 != nil || err2 != nil {
		return false
	}
	return baseURL.Hostname() == linkURL.Hostname()
}

// URLDetails holds the resolved information about a URL.
type URLDetails struct {
	InitialURL          string
	FinalURL            string
	StatusCode          int
	ContentType         string
	Filename            string
	IsRedirected        bool
	ErrorGettingDetails string
}

func ResolveURLDetails(initialURLStr string) (*URLDetails, error) {
	details := &URLDetails{InitialURL: initialURLStr}

	client := http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("HEAD", initialURLStr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HEAD request for '%s': %w", initialURLStr, err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HEAD request to '%s' (or subsequent redirects) failed: %w", initialURLStr, err)
	}
	defer func(Body io.ReadCloser) {
		if err := Body.Close(); err != nil {
			details.ErrorGettingDetails += fmt.Sprintf(" | failed to close response body: %v", err)
		}
	}(resp.Body)

	finalURL := resp.Request.URL
	details.FinalURL = finalURL.String()
	details.StatusCode = resp.StatusCode
	details.IsRedirected = initialURLStr != details.FinalURL

	contentTypeHeader := resp.Header.Get("Content-Type")
	if contentTypeHeader != "" {
		mediaType, _, parseErr := mime.ParseMediaType(contentTypeHeader)
		if parseErr == nil {
			details.ContentType = mediaType
		} else {
			details.ContentType = contentTypeHeader
			details.ErrorGettingDetails += fmt.Sprintf(" | failed to parse Content-Type header '%s': %v", contentTypeHeader, parseErr)
		}
	}

	var determinedFilename string
	contentDispositionHeader := resp.Header.Get("Content-Disposition")
	if contentDispositionHeader != "" {
		_, params, parseErr := mime.ParseMediaType(contentDispositionHeader)
		if parseErr == nil {
			if fn, ok := params["filename"]; ok && fn != "" {
				determinedFilename = fn
			}
		}
		if determinedFilename == "" && parseErr != nil {
			details.ErrorGettingDetails += fmt.Sprintf(" | failed to parse Content-Disposition header '%s': %v", contentDispositionHeader, parseErr)
		}
	}

	if determinedFilename == "" && details.StatusCode == http.StatusOK {
		if finalURL.Path != "" && finalURL.Path != "/" {
			base := path.Base(finalURL.Path)
			if base != "." && base != "/" && base != "" {
				unescapedBase, unescapeErr := url.PathUnescape(base)
				if unescapeErr == nil {
					determinedFilename = unescapedBase
				} else {
					determinedFilename = base
					details.ErrorGettingDetails += fmt.Sprintf(" | failed to unescape path base '%s': %v", base, unescapeErr)
				}
			}
		}
	}
	details.Filename = determinedFilename

	return details, nil
}
