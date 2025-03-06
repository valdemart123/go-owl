// Package browsers provides support for launching and managing
// different web browsers in the UI automation framework Owl Go.
package browsers

import (
	"errors"
	"fmt"
	"log"

	"github.com/valdemart123/go-owl/config"
)

// Browser interface to unify browser handling
type Browser interface {
	Launch() error
	Close() error
	OpenURL(url string) error
}

// GetBrowser initializes the correct browser based on JSON config
func GetBrowser() (Browser, error) {
	browserType := config.LoadBrowserType()
	log.Printf("Selected browser: %s\n", browserType)

	switch browserType {
	case "chrome":
		browser := &Chrome{}
		if err := browser.Launch(); err != nil {
			return nil, err
		}
		return browser, nil
	case "firefox":
		browser := &Firefox{}
		if err := browser.Launch(); err != nil {
			return nil, err
		}
		return browser, nil
	case "webkit":
		browser := &WebKit{}
		if err := browser.Launch(); err != nil {
			return nil, err
		}
		return browser, nil
	default:
		return nil, errors.New(fmt.Sprintf("Unsupported browser type: %s", browserType))
	}
}