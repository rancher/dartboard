package imageutils

import (
	"context"
	"errors"
	"image"
	"image/png"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	log "github.com/sirupsen/logrus"
)

func loadImageFromURL(url string) (image.Image, error) {
	//Get the response bytes from the url
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return nil, errors.New("received non-200 response code")
	}

	img, _, err := image.Decode(response.Body)
	if err != nil {
		return nil, err
	}

	return img, nil
}

// url must be publicly reachable
// imgFilePath must end in ".png"
func LocalPNGFromURL(url string, imgFilePath string) error {
	img, err := loadImageFromURL(url)
	if err != nil {
		log.Infof("Error loading image from URL (%s): %v\n", url, err)
		return err
	}
	imgFile, err := os.Create(imgFilePath)
	if err != nil {
		log.Infof("Error creating image file (%s): %v\n", imgFile.Name(), err)
		return err
	}
	defer imgFile.Close()
	if err = png.Encode(imgFile, img); err != nil {
		log.Infof("failed to encode: %v", err)
		return err
	}
	return nil
}

func HttpCookiesToSlice(cookies []*http.Cookie, list *[]string) {
	for _, cookie := range cookies {
		*list = append(*list, cookie.Name, cookie.Value)
	}
}

func setcookies(url string, cookies ...string) chromedp.Tasks {
	uri := strings.Split(url, "//")[1]
	domain := strings.Split(uri, "/")[0]
	return chromedp.Tasks{
		chromedp.ActionFunc(func(ctx context.Context) error {
			// create cookie expiration
			expr := cdp.TimeSinceEpoch(time.Now().Add(180 * 24 * time.Hour))
			// add cookies to chrome
			for i := 0; i < len(cookies); i += 2 {
				err := network.SetCookie(cookies[i], cookies[i+1]).
					WithExpires(&expr).
					WithDomain(domain).
					WithHTTPOnly(true).
					Do(ctx)
				if err != nil {
					return err
				}
			}
			return nil
		}),
		// navigate to site
		chromedp.Navigate(url),
	}
}

// fullScreenshot takes a screenshot of the entire browser viewport.
//
// Note: chromedp.FullScreenshot overrides the device's emulation settings. Use
// device.Reset to reset the emulation and viewport settings.
func fullScreenshot(urlstr string, quality int, res *[]byte) chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.Navigate(urlstr),
		chromedp.FullScreenshot(res, quality),
	}
}

func enableLifeCycleEvents() chromedp.ActionFunc {
	return func(ctx context.Context) error {
		err := page.Enable().Do(ctx)
		if err != nil {
			return err
		}
		err = page.SetLifecycleEventsEnabled(true).Do(ctx)
		if err != nil {
			return err
		}
		return nil
	}
}

// waitFor blocks until eventName is received.
// Examples of events you can wait for:
//
//	init, DOMContentLoaded, firstPaint,
//	firstContentfulPaint, firstImagePaint,
//	firstMeaningfulPaintCandidate,
//	load, networkAlmostIdle, firstMeaningfulPaint, networkIdle
//
// This is not super reliable, I've already found incidental cases where
// networkIdle was sent before load. It's probably smart to see how
// puppeteer implements this exactly.
// sourced from https://github.com/chromedp/chromedp/issues/431#issuecomment-592950397
func waitFor(ctx context.Context, eventName string) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		ch := make(chan struct{})
		cctx, cancel := context.WithCancel(ctx)
		chromedp.ListenTarget(cctx, func(ev interface{}) {
			switch e := ev.(type) {
			case *page.EventLifecycleEvent:
				if e.Name == eventName {
					cancel()
					close(ch)
				}
			}
		})
		select {
		case <-ch:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func URLScreenshotToPNG(url string, imgFilePath string, sel string, cookies ...string) error {
	opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.WindowSize(1920, 1200))
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()
	// also set up a custom logger
	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()
	var err error
	var buf []byte
	// capture entire browser viewport, returning png with quality=100
	if len(cookies)%2 != 0 {
		log.Fatal("Length of cookies must be divisible by 2. Proceeding without setting cookies.")
		if err = chromedp.Run(ctx, fullScreenshot(url, 100, &buf)); err != nil {
			log.Fatal(err)
			return err
		}
	} else {
		log.Infof("Length of cookies is divisible by 2. Proceeding with selecting the proper Action Chain")
		if sel != "" {
			log.Infof("Selector was provided, proceeding with WaitVisible Action Chain")
			if err = chromedp.Run(ctx, setcookies(url, cookies...), chromedp.Tasks{
				enableLifeCycleEvents(),
				waitFor(ctx, "networkIdle"),
			}, chromedp.WaitVisible(sel), chromedp.FullScreenshot(&buf, 100)); err != nil {
				log.Fatal(err)
				return err
			}
		} else {
			log.Infof("Selector was not provided, proceeding with Sleep Action Chain of 5 seconds")
			if err = chromedp.Run(ctx, setcookies(url, cookies...), chromedp.Sleep(time.Second*5), chromedp.FullScreenshot(&buf, 100)); err != nil {
				log.Fatal(err)
				return err
			}
		}
	}
	if f, err := os.Create(imgFilePath); err == nil {
		if _, err := f.Write(buf); err != nil {
			log.Fatal(err)
			return err
		}
	}
	return nil
}
