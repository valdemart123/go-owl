package browsers

import (
	"errors"
	"log"

	"github.com/go-rod/rod"
)

// Chrome struct using Rod
type Chrome struct {
	Browser *rod.Browser
	Page    *rod.Page
}

// Launch starts a new Chrome browser instance
func (c *Chrome) Launch() error {
	log.Println("Launching Chrome...")
	c.Browser = rod.New().MustConnect()
	return nil
}

// Close shuts down the Chrome browser instance
func (c *Chrome) Close() error {
	if c.Browser != nil {
		c.Browser.MustClose()
		log.Println("Chrome browser closed successfully.")
	}
	return nil
}

func (c *Chrome) OpenURL(url string) error {
	if c.Browser == nil {
		return errors.New("browser not launched")
	}
	c.Page = c.Browser.MustPage(url)
	return nil
}
