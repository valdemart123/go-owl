package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// setup installs necessary browser drivers and updates them if needed.
func setup() {
	fmt.Println("Setting up Owl Automation Framework...")

	installChromeDriver()
	installFirefoxDriver()
	installWebkitDriver()

	fmt.Println("Setup complete. Run your tests with `go test ./tests -v`.")
}

// installChromeDriver ensures Chrome is installed and up to date.
func installChromeDriver() {
	fmt.Println("Checking Chrome version...")

	installedVersion := getInstalledChromeVersion()
	latestVersion := getLatestChromeVersion()

	if installedVersion == "Not Installed" {
		log.Fatal("Chrome is not installed. Please install Chrome manually.")
	}

	if installedVersion == latestVersion {
		fmt.Println("Chrome is up to date.")
		return
	}

	fmt.Printf("Updating Chrome (Installed: %s, Latest: %s)...\n", installedVersion, latestVersion)
	updateChrome()
}

// getInstalledChromeVersion retrieves the installed Chrome version.
func getInstalledChromeVersion() string {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("/Applications/Google Chrome.app/Contents/MacOS/Google Chrome", "--version")
	case "linux":
		cmd = exec.Command("google-chrome", "--version")
	case "windows":
		cmd = exec.Command("powershell", "(Get-Item 'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe').VersionInfo.FileVersion")
	default:
		log.Fatal("Unsupported OS for Chrome detection.")
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "Not Installed"
	}

	version := strings.TrimSpace(string(output))
	parts := strings.Fields(version)
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}

	return "Unknown"
}

// getLatestChromeVersion fetches the latest stable Chrome version from Google's API.
func getLatestChromeVersion() string {
	cmd := exec.Command("sh", "-c", "curl -s https://chromedriver.storage.googleapis.com/LATEST_RELEASE")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal("Failed to fetch the latest Chrome version:", err)
	}
	return strings.TrimSpace(string(output))
}

// updateChrome attempts to update Chrome on supported platforms.
func updateChrome() {
	switch runtime.GOOS {
	case "darwin":
		fmt.Println("To update Chrome on macOS, download it from https://www.google.com/chrome/")
	case "linux":
		cmd := exec.Command("sh", "-c", "sudo apt update && sudo apt install --only-upgrade google-chrome-stable")
		if err := cmd.Run(); err != nil {
			log.Fatal("Failed to update Chrome:", err)
		}
	case "windows":
		fmt.Println("To update Chrome on Windows, visit https://www.google.com/chrome/")
	default:
		log.Fatal("Unsupported OS for automatic Chrome updates.")
	}
}

// installFirefoxDriver automatically downloads and installs/upgrades Geckodriver.
func installFirefoxDriver() {
	fmt.Println("Checking Geckodriver version...")

	installedVersion := getInstalledGeckoDriverVersion()
	latestVersion := getLatestGeckoDriverVersion()

	if installedVersion == latestVersion {
		fmt.Println("Geckodriver is up to date.")
		return
	}

	fmt.Printf("Updating Geckodriver (Installed: %s, Latest: %s)...\n", installedVersion, latestVersion)
	downloadAndInstallGeckoDriver(latestVersion)
	fmt.Println("Geckodriver updated successfully.")
}

// installWebkitDriver enables Safari WebDriver (only for macOS).
func installWebkitDriver() {
	if runtime.GOOS != "darwin" {
		fmt.Println("WebKit (Safari) automation is only supported on macOS.")
		return
	}

	fmt.Println("Enabling Safari WebDriver...")
	cmd := exec.Command("safaridriver", "--enable")
	if err := cmd.Run(); err != nil {
		log.Fatal("Failed to enable Safari WebDriver:", err)
	}

	fmt.Println("Safari WebDriver enabled.")
}

// getInstalledGeckoDriverVersion checks the installed version of Geckodriver.
func getInstalledGeckoDriverVersion() string {
	cmd := exec.Command("geckodriver", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "Not Installed"
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		return strings.Fields(lines[0])[1] // Extract version number from output
	}
	return "Unknown"
}

// getLatestGeckoDriverVersion fetches the latest Geckodriver version from GitHub.
func getLatestGeckoDriverVersion() string {
	cmd := exec.Command("sh", "-c", "curl -s https://api.github.com/repos/mozilla/geckodriver/releases/latest | grep 'tag_name' | cut -d '\"' -f 4")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal("Failed to fetch latest Geckodriver version:", err)
	}
	return strings.TrimSpace(string(output))
}

// downloadAndInstallGeckoDriver downloads and installs the latest Geckodriver version.
func downloadAndInstallGeckoDriver(version string) {
	var downloadURL, outputFile string
	switch runtime.GOOS {
	case "darwin":
		downloadURL = fmt.Sprintf("https://github.com/mozilla/geckodriver/releases/download/%s/geckodriver-macos.tar.gz", version)
		outputFile = "/usr/local/bin/geckodriver"
	case "linux":
		downloadURL = fmt.Sprintf("https://github.com/mozilla/geckodriver/releases/download/%s/geckodriver-linux64.tar.gz", version)
		outputFile = "/usr/local/bin/geckodriver"
	case "windows":
		downloadURL = fmt.Sprintf("https://github.com/mozilla/geckodriver/releases/download/%s/geckodriver-win64.zip", version)
		outputFile = os.Getenv("USERPROFILE") + "\\bin\\geckodriver.exe"
	default:
		log.Fatal("Unsupported OS for Geckodriver installation.")
	}

	// Download the driver
	fmt.Println("Downloading Geckodriver...")
	if err := downloadAndExtract(downloadURL, outputFile); err != nil {
		log.Fatalf("Failed to install Geckodriver: %v", err)
	}
}

// downloadAndExtract downloads a file and extracts it if needed.
func downloadAndExtract(url, outputPath string) error {
	cmd := exec.Command("sh", "-c", fmt.Sprintf("curl -L %s -o /tmp/driver && tar -xzf /tmp/driver -C /usr/local/bin && chmod +x %s", url, outputPath))
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: owl <command>\nAvailable commands:\n  setup  Install or update required browser drivers")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "setup":
		setup()
	default:
		fmt.Println("Unknown command:", os.Args[1])
		os.Exit(1)
	}
}