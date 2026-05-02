package api

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"resty.dev/v3"
)

type ReleaseInfo struct {
	TagName string
	Version string
	URL     string
}

func CheckForUpdate(currentVersion string) (*ReleaseInfo, error) {
	if currentVersion == "dev" || currentVersion == "" {
		return nil, nil
	}

	owner := os.Getenv("VAR_CLI_OWNER")
	if owner == "" {
		owner = "json-nan"
	}
	repo := os.Getenv("VAR_CLI_REPO")
	if repo == "" {
		repo = "var-cli"
	}

	apiBase := os.Getenv("VAR_CLI_API_URL")
	if apiBase == "" {
		apiBase = "https://api.github.com"
	}
	downloadBase := os.Getenv("VAR_CLI_DOWNLOAD_URL")
	if downloadBase == "" {
		downloadBase = "https://github.com"
	}

	r := resty.New()
	r.SetTimeout(10 * time.Second)

	var result struct {
		TagName string `json:"tag_name"`
	}
	apiURL := fmt.Sprintf("%s/repos/%s/%s/releases/latest", apiBase, owner, repo)
	resp, err := r.NewRequest().SetResult(&result).Get(apiURL)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("github api returned %s", resp.Status())
	}

	tagVersion := strings.TrimPrefix(result.TagName, "v")
	current := strings.TrimPrefix(currentVersion, "v")

	if !isNewerVersion(tagVersion, current) {
		return nil, nil
	}

	osName := runtime.GOOS
	arch := runtime.GOARCH
	switch arch {
	case "amd64":
		arch = "x86_64"
	case "386":
		arch = "i386"
	}

	ext := "tar.gz"
	if osName == "windows" {
		ext = "zip"
	}

	// GitHub release filename: var-cli_0.2.0_Darwin_arm64.tar.gz (NO "v" prefix)
	filename := fmt.Sprintf("var-cli_%s_%s_%s.%s", tagVersion, strings.Title(osName), arch, ext)
	url := fmt.Sprintf("%s/%s/%s/releases/download/%s/%s", downloadBase, owner, repo, result.TagName, filename)

	return &ReleaseInfo{
		TagName: result.TagName,
		Version: tagVersion,
		URL:     url,
	}, nil
}

func isNewerVersion(latest, current string) bool {
	lp := strings.Split(latest, ".")
	cp := strings.Split(current, ".")
	for i := 0; i < len(lp) && i < len(cp); i++ {
		var lv, cv int
		fmt.Sscanf(lp[i], "%d", &lv)
		fmt.Sscanf(cp[i], "%d", &cv)
		if lv > cv {
			return true
		}
		if lv < cv {
			return false
		}
	}
	return len(lp) > len(cp)
}

func ApplyUpdate(url string) error {
	r := resty.New()
	r.SetTimeout(120 * time.Second)

	resp, err := r.NewRequest().Get(url)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("download failed: %s", resp.Status())
	}

	tmpFile, err := os.CreateTemp("", "var-cli-update-*")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()

	if err := os.WriteFile(tmpPath, resp.Bytes(), 0644); err != nil {
		return err
	}
	defer os.Remove(tmpPath)

	execPath, err := os.Executable()
	if err != nil {
		return err
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return err
	}

	var newBinaryPath string
	if strings.HasSuffix(url, ".zip") {
		newBinaryPath, err = extractZip(tmpPath, "var-cli")
	} else {
		newBinaryPath, err = extractTarGz(tmpPath, "var-cli")
	}
	if err != nil {
		return fmt.Errorf("extraction failed: %w", err)
	}
	defer os.Remove(newBinaryPath)

	info, err := os.Stat(newBinaryPath)
	if err != nil {
		return err
	}

	if runtime.GOOS == "windows" {
		oldPath := execPath + ".old"
		_ = os.Remove(oldPath)
		if err := os.Rename(execPath, oldPath); err != nil {
			return fmt.Errorf("failed to backup current binary: %w", err)
		}
		if err := os.Rename(newBinaryPath, execPath); err != nil {
			_ = os.Rename(oldPath, execPath)
			return fmt.Errorf("failed to install new binary: %w", err)
		}
	} else {
		if err := os.Rename(newBinaryPath, execPath); err != nil {
			return fmt.Errorf("failed to install new binary: %w", err)
		}
		if err := os.Chmod(execPath, info.Mode()|0111); err != nil {
			return err
		}
	}

	return nil
}

func extractTarGz(path, binaryName string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if hdr.Name == binaryName || filepath.Base(hdr.Name) == binaryName {
			tmpOut, err := os.CreateTemp("", "var-cli-bin-*")
			if err != nil {
				return "", err
			}
			if _, err := io.Copy(tmpOut, tr); err != nil {
				tmpOut.Close()
				return "", err
			}
			tmpOut.Close()
			os.Chmod(tmpOut.Name(), hdr.FileInfo().Mode()|0111)
			return tmpOut.Name(), nil
		}
	}
	return "", fmt.Errorf("binary %s not found in archive", binaryName)
}

func extractZip(path, binaryName string) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == binaryName || filepath.Base(f.Name) == binaryName {
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			defer rc.Close()

			tmpOut, err := os.CreateTemp("", "var-cli-bin-*")
			if err != nil {
				return "", err
			}
			if _, err := io.Copy(tmpOut, rc); err != nil {
				tmpOut.Close()
				return "", err
			}
			tmpOut.Close()
			os.Chmod(tmpOut.Name(), f.Mode()|0111)
			return tmpOut.Name(), nil
		}
	}
	return "", fmt.Errorf("binary %s not found in archive", binaryName)
}
