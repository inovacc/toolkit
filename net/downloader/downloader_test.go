package downloader

import "testing"

func TestDownloader(t *testing.T) {
	base := "https://app.unpkg.com/@excalidraw/excalidraw@0.18.0"
	outDir := "downloads"

	d := NewDownloader()
	d.Include(base) // se asume que este es el punto de partida
	d.base = base   // necesario para validar dominio y resolver URL

	d.Crawler()

	// Mostrar links encontrados
	for _, link := range d.List() {
		println("📦", link)
	}

	d.Download(outDir)
}
