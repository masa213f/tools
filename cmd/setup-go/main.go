package main

import (
	"archive/tar"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/masa213f/tools/pkg/util"
)

const (
	installPrefix = "/usr/local/"
	cacheDir      = "/var/cache/setup-go"
)

func downloadFileName(version string) string {
	return "go" + version + ".linux-amd64.tar.gz"
}

func downloadURL(version string) string {
	return "https://dl.google.com/go/" + downloadFileName(version)
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

	downloadFilePath := filepath.Join(cacheDir, downloadFileName(goVersion))
	if !util.FileExists(downloadFilePath) {
		err := util.DownloadFile(downloadURL(goVersion), downloadFilePath)
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

	err = ExtractTarGz(downloadFilePath, installPrefix)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to extract file: %v\n", err)
		os.Exit(1)
	}
}

func ExtractTarGz(tgzfilePath, outputDir string) error {
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
			{
				outFile, err := os.Create(filePath)
				if err != nil {
					return fmt.Errorf("failed to create file %s; %w", filePath, err)
				}
				defer outFile.Close()
				if _, err := io.Copy(outFile, tarReader); err != nil {
					return fmt.Errorf("failed to write file %s; %w", filePath, err)
				}
				if err := os.Chmod(filePath, fileMode); err != nil {
					return fmt.Errorf("failed to change mode %s; %w", filePath, err)
				}
			}
		default:
			return fmt.Errorf("unknown type: %s %d", header.Name, header.Typeflag)
		}
	}
	return nil
}
