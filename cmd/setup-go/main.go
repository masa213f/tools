package main

import (
	"archive/tar"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const (
	installPrefix = "/usr/local/"
	cacheDir      = "/var/cache/setup-go"
	urlPrefix     = "https://dl.google.com/go/"
)

func goArchiveFile(version string) string {
	return "go" + version + ".linux-amd64.tar.gz"
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func download(url, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s; %w", filePath, err)
	}
	defer file.Close()

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download file %s; %w", filePath, err)
	}
	defer resp.Body.Close()

	size, err := io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file %s; %w", filePath, err)
	}

	fmt.Printf("downloaded file: %s (size %d)\n", filePath, size)
	return err
}

func dump(r io.Reader, filePath string, mode os.FileMode) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s; %w", filePath, err)
	}
	defer file.Close()

	_, err = io.Copy(file, r)
	if err != nil {
		return fmt.Errorf("failed to write file %s; %w", filePath, err)
	}

	err = os.Chmod(filePath, mode)
	if err != nil {
		return fmt.Errorf("failed to change mode %s; %w", filePath, err)
	}

	return nil
}

func extractTarGz(tgzfilePath, outputDir string) error {
	file, err := os.Open(tgzfilePath)
	if err != nil {
		return fmt.Errorf("failed to open file; %w", err)
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader; %w", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to get next reader; %w", err)
		}

		filePath := filepath.Join(outputDir, header.Name)
		fileMode := os.FileMode(header.Mode)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.Mkdir(filePath, fileMode); err != nil {
				return fmt.Errorf("failed to create directory %s; %w", filePath, err)
			}
		case tar.TypeReg:
			if err := dump(tarReader, filePath, fileMode); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown type: %s %d", header.Name, header.Typeflag)
		}
	}
	return nil
}

func main() {
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Println("Usage: setup-go GO_VERSION")
		os.Exit(1)
	}
	goVersion := flag.Arg(0)

	err := os.Mkdir(cacheDir, 0775)
	if err != nil && !os.IsExist(err) {
		fmt.Fprintf(os.Stderr, "failed to create cache directory; %v\n", err)
		os.Exit(1)
	}

	downloadedFile := filepath.Join(cacheDir, goArchiveFile(goVersion))
	if !fileExists(downloadedFile) {
		err := download(urlPrefix+goArchiveFile(goVersion), downloadedFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to download file: %v\n", err)
			os.Exit(1)
		}
	}

	removeDir := filepath.Join(installPrefix, "go")
	err = os.RemoveAll(removeDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to remove directory; %v\n", err)
		os.Exit(1)
	}

	err = extractTarGz(downloadedFile, installPrefix)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to extract file: %v\n", err)
		os.Exit(1)
	}
}
