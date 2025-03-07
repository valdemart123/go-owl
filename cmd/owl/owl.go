package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// Run starts the CLI tool
func Run() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: owl <command>\nAvailable commands:\n  setup  Install or update required browser drivers")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "setup":
		if err := setup(); err != nil {
			fmt.Printf("Setup encountered errors: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Println("Unknown command:", os.Args[1])
		os.Exit(1)
	}
}

// setup installs necessary browser drivers and updates them if needed.
func setup() error {
	fmt.Println("Setting up Owl Automation Framework...")
	
	var setupErrors []string
	
	if err := installChromeDriver(); err != nil {
		setupErrors = append(setupErrors, fmt.Sprintf("Chrome driver: %v", err))
	}
	
	if err := installFirefoxDriver(); err != nil {
		setupErrors = append(setupErrors, fmt.Sprintf("Firefox driver: %v", err))
	}
	
	if err := installWebkitDriver(); err != nil {
		setupErrors = append(setupErrors, fmt.Sprintf("Webkit driver: %v", err))
	}
	
	if len(setupErrors) > 0 {
		fmt.Println("Setup completed with some issues:")
		for _, err := range setupErrors {
			fmt.Printf("- %s\n", err)
		}
	} else {
		fmt.Println("Setup completed successfully.")
	}
	
	fmt.Println("Run your tests with `go test ./tests -v`.")
	
	if len(setupErrors) > 0 {
		return fmt.Errorf("setup completed with %d issues", len(setupErrors))
	}
	return nil
}

// installChromeDriver ensures Chrome is installed and up to date.
func installChromeDriver() error {
	fmt.Println("Checking Chrome version...")

	installedVersion, err := getInstalledChromeVersion()
	if err != nil {
		return fmt.Errorf("failed to get installed Chrome version: %w", err)
	}

	if installedVersion == "" {
		return errors.New("Chrome is not installed. Please install Chrome manually")
	}

	latestVersion, err := getLatestChromeDriverVersion()
	if err != nil {
		return fmt.Errorf("failed to get latest Chrome driver version: %w", err)
	}

	fmt.Printf("Chrome version: %s, Latest ChromeDriver: %s\n", installedVersion, latestVersion)

	// Get major version to match with ChromeDriver
	re := regexp.MustCompile(`^(\d+)\.`)
	installedMajor := re.FindStringSubmatch(installedVersion)
	if len(installedMajor) < 2 {
		return fmt.Errorf("failed to parse Chrome version: %s", installedVersion)
	}

	// Check if we need to download the appropriate ChromeDriver
	chromeDriverPath, err := getChromeDriverPath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(chromeDriverPath); os.IsNotExist(err) {
		// ChromeDriver not installed
		fmt.Println("ChromeDriver not found. Installing...")
		return downloadAndInstallChromeDriver(installedMajor[1])
	}

	// TODO: Check if the installed ChromeDriver version matches the Chrome version
	// This would require parsing the ChromeDriver version and comparing major versions
	fmt.Println("ChromeDriver is already installed. Use 'owl setup --force' to reinstall drivers.")
	return nil
}

// getInstalledChromeVersion retrieves the installed Chrome version.
func getInstalledChromeVersion() (string, error) {
	var cmd *exec.Cmd
	var output []byte

	switch runtime.GOOS {
	case "darwin":
		chromePaths := []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chrome.app/Contents/MacOS/Chrome",
		}
		
		for _, path := range chromePaths {
			if _, err := os.Stat(path); err == nil {
				cmd = exec.Command(path, "--version")
				output, err = cmd.CombinedOutput()
				if err == nil {
					break
				}
			}
		}
		
	case "linux":
		possibleCommands := []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser"}
		
		for _, browser := range possibleCommands {
			cmd = exec.Command("which", browser)
			if err := cmd.Run(); err == nil {
				cmd = exec.Command(browser, "--version")
				output, err = cmd.CombinedOutput()
				if err == nil {
					break
				}
			}
		}
		
	case "windows":
		// Try common installation paths
		chromePaths := []string{
			filepath.Join(os.Getenv("ProgramFiles"), "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(os.Getenv("LocalAppData"), "Google", "Chrome", "Application", "chrome.exe"),
		}
		
		for _, path := range chromePaths {
			if _, err := os.Stat(path); err == nil {
				// Found Chrome, now get its version
				cmd = exec.Command("powershell", "-Command", fmt.Sprintf("(Get-Item '%s').VersionInfo.FileVersion", path))
				output, err = cmd.CombinedOutput()
				if err == nil {
					break
				}
			}
		}
		
	default:
		return "", fmt.Errorf("unsupported OS for Chrome detection: %s", runtime.GOOS)
	}
	
	version := strings.TrimSpace(string(output))
	if version == "" {
		return "", errors.New("Chrome not found")
	}
	// Extract version number from output (format varies by OS)
	re := regexp.MustCompile(`\d+\.\d+\.\d+\.\d+`)
	match := re.FindString(version)
	if match != "" {
		return match, nil
	}

	// Try a more lenient pattern if the strict one fails
	re = regexp.MustCompile(`\d+\.\d+\.\d+`)
	match = re.FindString(version)
	if match != "" {
		return match, nil
	}

	return "", fmt.Errorf("failed to parse Chrome version from: %s", version)
}

// getLatestChromeDriverVersion fetches the latest stable ChromeDriver version.
func getLatestChromeDriverVersion() (string, error) {
	resp, err := http.Get("https://chromedriver.storage.googleapis.com/LATEST_RELEASE")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(body)), nil
}

// getChromeDriverPath returns the platform-specific path for ChromeDriver.
func getChromeDriverPath() (string, error) {
	execName := "chromedriver"
	if runtime.GOOS == "windows" {
		execName = "chromedriver.exe"
	}

	// Check if chromedriver is in PATH
	path, err := exec.LookPath(execName)
	if err == nil {
		return path, nil
	}

	// Use standard locations based on OS
	var binPath string
	switch runtime.GOOS {
	case "darwin", "linux":
		// Check /usr/local/bin and /usr/bin
		for _, dir := range []string{"/usr/local/bin", "/usr/bin"} {
			path = filepath.Join(dir, execName)
			if _, err := os.Stat(path); err == nil {
				return path, nil
			}
		}
		
		// Default location for installation if not found
		binPath = "/usr/local/bin"
	case "windows":
		// Use %USERPROFILE%\bin or create it
		binPath = filepath.Join(os.Getenv("USERPROFILE"), "bin")
	default:
		return "", fmt.Errorf("unsupported OS for ChromeDriver: %s", runtime.GOOS)
	}

	// Ensure bin directory exists
	if err := os.MkdirAll(binPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create bin directory: %w", err)
	}

	return filepath.Join(binPath, execName), nil
}

// downloadAndInstallChromeDriver downloads and installs ChromeDriver for the given Chrome version.
func downloadAndInstallChromeDriver(chromeMajorVersion string) error {
	// Get the latest driver version for this Chrome major version
	url := fmt.Sprintf("https://chromedriver.storage.googleapis.com/LATEST_RELEASE_%s", chromeMajorVersion)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to get ChromeDriver version: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	driverVersion := strings.TrimSpace(string(body))

	// Determine platform-specific download URL
	var platform string
	switch runtime.GOOS {
	case "darwin":
		if runtime.GOARCH == "arm64" {
			platform = "mac_arm64"
		} else {
			platform = "mac64"
		}
	case "linux":
		platform = "linux64"
	case "windows":
		platform = "win32"
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	downloadURL := fmt.Sprintf("https://chromedriver.storage.googleapis.com/%s/chromedriver_%s.zip", driverVersion, platform)
	fmt.Printf("Downloading ChromeDriver %s for Chrome %s from %s\n", driverVersion, chromeMajorVersion, downloadURL)

	// Create temporary directory for download
	tempDir, err := os.MkdirTemp("", "chromedriver")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	// Download the file
	archivePath := filepath.Join(tempDir, "chromedriver.zip")
	if err := downloadFile(downloadURL, archivePath); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	// Get destination path
	driverPath, err := getChromeDriverPath()
	if err != nil {
		return err
	}

	// Extract the zip file
	if err := extractZip(archivePath, tempDir); err != nil {
		return fmt.Errorf("extraction failed: %w", err)
	}

	// Move the driver to the final location
	srcDriver := filepath.Join(tempDir, "chromedriver")
	if runtime.GOOS == "windows" {
		srcDriver += ".exe"
	}

	// Make sure the target directory exists
	targetDir := filepath.Dir(driverPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Copy the file
	if err := copyFile(srcDriver, driverPath); err != nil {
		return fmt.Errorf("failed to install driver: %w", err)
	}

	// Make it executable on Unix systems
	if runtime.GOOS != "windows" {
		if err := os.Chmod(driverPath, 0755); err != nil {
			return fmt.Errorf("failed to make driver executable: %w", err)
		}
	}

	fmt.Printf("ChromeDriver %s installed successfully at %s\n", driverVersion, driverPath)
	return nil
}

// installFirefoxDriver installs or updates Geckodriver.
func installFirefoxDriver() error {
	fmt.Println("Checking Geckodriver version...")

	installedVersion, err := getInstalledGeckoDriverVersion()
	if err != nil {
		// If error is just that driver is not installed, continue to installation
		if !strings.Contains(err.Error(), "not installed") {
			return fmt.Errorf("failed to check installed Geckodriver: %w", err)
		}
	}

	latestVersion, err := getLatestGeckoDriverVersion()
	if err != nil {
		return fmt.Errorf("failed to get latest Geckodriver version: %w", err)
	}

	// Skip if already up to date
	if installedVersion == latestVersion {
		fmt.Printf("Geckodriver is up to date (version %s)\n", installedVersion)
		return nil
	}

	fmt.Printf("Updating Geckodriver (Installed: %s, Latest: %s)...\n", installedVersion, latestVersion)
	if err := downloadAndInstallGeckoDriver(latestVersion); err != nil {
		return fmt.Errorf("failed to install Geckodriver: %w", err)
	}

	fmt.Println("Geckodriver updated successfully.")
	return nil
}

// getInstalledGeckoDriverVersion checks the installed version of Geckodriver.
func getInstalledGeckoDriverVersion() (string, error) {
	geckoPath, err := exec.LookPath("geckodriver")
	if err != nil {
		return "", fmt.Errorf("geckodriver not installed: %w", err)
	}

	cmd := exec.Command(geckoPath, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get geckodriver version: %w", err)
	}

	// Parse version from output
	re := regexp.MustCompile(`geckodriver (\d+\.\d+\.\d+)`)
	matches := re.FindStringSubmatch(string(output))
	if len(matches) < 2 {
		return "", fmt.Errorf("failed to parse geckodriver version from: %s", string(output))
	}

	return matches[1], nil
}

// getLatestGeckoDriverVersion fetches the latest Geckodriver version from GitHub.
func getLatestGeckoDriverVersion() (string, error) {
	resp, err := http.Get("https://api.github.com/repos/mozilla/geckodriver/releases/latest")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	// Remove 'v' prefix if present
	version := release.TagName
	if strings.HasPrefix(version, "v") {
		version = version[1:]
	}

	return version, nil
}

// installWebkitDriver enables Safari WebDriver (only for macOS).
func installWebkitDriver() error {
	if runtime.GOOS != "darwin" {
		fmt.Println("WebKit (Safari) automation is only supported on macOS. Skipping.")
		return nil
	}

	fmt.Println("Checking Safari WebDriver...")

	// Check if safaridriver is available
	_, err := exec.LookPath("safaridriver")
	if err != nil {
		return fmt.Errorf("safaridriver not found: %w", err)
	}

	fmt.Println("Enabling Safari WebDriver...")
	cmd := exec.Command("safaridriver", "--enable")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if it failed due to permissions
		if strings.Contains(string(output), "administrator privileges") {
			fmt.Println("Safari WebDriver requires administrator privileges to enable.")
			fmt.Println("Please run the following command manually in Terminal:")
			fmt.Println("  sudo safaridriver --enable")
			return nil
		}
		return fmt.Errorf("failed to enable Safari WebDriver: %w, output: %s", err, string(output))
	}

	fmt.Println("Safari WebDriver enabled successfully.")
	return nil
}

// downloadAndInstallGeckoDriver downloads and installs the latest Geckodriver version.
func downloadAndInstallGeckoDriver(version string) error {
	// Determine the correct download URL based on platform
	var downloadURL string
	var archiveExt string

	switch runtime.GOOS {
	case "darwin":
		if runtime.GOARCH == "arm64" {
			downloadURL = fmt.Sprintf("https://github.com/mozilla/geckodriver/releases/download/v%s/geckodriver-v%s-macos-aarch64.tar.gz", version, version)
		} else {
			downloadURL = fmt.Sprintf("https://github.com/mozilla/geckodriver/releases/download/v%s/geckodriver-v%s-macos.tar.gz", version, version)
		}
		archiveExt = ".tar.gz"
	case "linux":
		if runtime.GOARCH == "arm64" {
			downloadURL = fmt.Sprintf("https://github.com/mozilla/geckodriver/releases/download/v%s/geckodriver-v%s-linux-aarch64.tar.gz", version, version)
		} else {
			downloadURL = fmt.Sprintf("https://github.com/mozilla/geckodriver/releases/download/v%s/geckodriver-v%s-linux64.tar.gz", version, version)
		}
		archiveExt = ".tar.gz"
	case "windows":
		downloadURL = fmt.Sprintf("https://github.com/mozilla/geckodriver/releases/download/v%s/geckodriver-v%s-win64.zip", version, version)
		archiveExt = ".zip"
	default:
		return fmt.Errorf("unsupported OS for Geckodriver installation: %s", runtime.GOOS)
	}

	// Create temporary directory for download
	tempDir, err := os.MkdirTemp("", "geckodriver")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	// Download the file
	archivePath := filepath.Join(tempDir, "geckodriver"+archiveExt)
	fmt.Printf("Downloading Geckodriver from: %s\n", downloadURL)
	if err := downloadFile(downloadURL, archivePath); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	// Extract the file
	fmt.Println("Extracting Geckodriver...")
	if strings.HasSuffix(archivePath, ".zip") {
		if err := extractZip(archivePath, tempDir); err != nil {
			return fmt.Errorf("extraction failed: %w", err)
		}
	} else {
		if err := extractTarGz(archivePath, tempDir); err != nil {
			return fmt.Errorf("extraction failed: %w", err)
		}
	}

	// Get the destination path
	binPath := getBinDirectory()
	if err := os.MkdirAll(binPath, 0755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	// Geckodriver executable name
	driverName := "geckodriver"
	if runtime.GOOS == "windows" {
		driverName += ".exe"
	}

	srcPath := filepath.Join(tempDir, driverName)
	dstPath := filepath.Join(binPath, driverName)

	// Copy to destination
	if err := copyFile(srcPath, dstPath); err != nil {
		return fmt.Errorf("failed to install driver: %w", err)
	}

	// Make executable on Unix
	if runtime.GOOS != "windows" {
		if err := os.Chmod(dstPath, 0755); err != nil {
			return fmt.Errorf("failed to make driver executable: %w", err)
		}
	}

	fmt.Printf("Geckodriver v%s installed successfully at %s\n", version, dstPath)
	return nil
}

// getBinDirectory returns the appropriate bin directory for the current OS
func getBinDirectory() string {
	switch runtime.GOOS {
	case "darwin", "linux":
		// Check if we have write access to /usr/local/bin
		if err := os.MkdirAll("/usr/local/bin", 0755); err == nil {
			return "/usr/local/bin"
		}
		// Fallback to user's home directory
		homeDir, err := os.UserHomeDir()
		if err == nil {
			binDir := filepath.Join(homeDir, "bin")
			os.MkdirAll(binDir, 0755)
			return binDir
		}
		// Last resort, use current directory
		return "."
	case "windows":
		binDir := filepath.Join(os.Getenv("USERPROFILE"), "bin")
		os.MkdirAll(binDir, 0755)
		return binDir
	default:
		return "."
	}
}

// downloadFile downloads a file from a URL to a local path
func downloadFile(url, outputPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	out, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// extractZip extracts a zip archive to the specified directory
func extractZip(zipPath, destDir string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, file := range reader.File {
		path := filepath.Join(destDir, file.Name)

		// Check for ZipSlip vulnerability
		if !strings.HasPrefix(path, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", path)
		}

		if file.FileInfo().IsDir() {
			os.MkdirAll(path, file.Mode())
			continue
		}

		// Create directory tree
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}

		// Create the file
		destFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}

		srcFile, err := file.Open()
		if err != nil {
			destFile.Close()
			return err
		}

		_, err = io.Copy(destFile, srcFile)
		srcFile.Close()
		destFile.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

// extractTarGz extracts a .tar.gz archive to the specified directory
func extractTarGz(tarGzPath, destDir string) error {
	file, err := os.Open(tarGzPath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		path := filepath.Join(destDir, header.Name)

		// Check for Tar Slip vulnerability
		if !strings.HasPrefix(path, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", path)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			// Create directory tree
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return err
			}

			outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}

	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	return destFile.Sync()
}