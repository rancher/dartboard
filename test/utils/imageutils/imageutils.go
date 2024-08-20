package imageutils

import (
	"context"
	"image"
	"image/png"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"

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

// filePath must end in ".html"
func LocalHTMLFromURL(url, filepath string) error {
	//Get the response bytes from the url
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return errors.Wrap(err, "LocalHTMLFromURL: received non-200 response code. ")
	}

	//open a file for writing
	file, err := os.Create(filepath)
	if err != nil {
		log.Fatal(err)
		return errors.Wrap(err, "LocalHTMLFromURL: failed to create file")
	}
	defer file.Close()

	// Use io.Copy to just dump the response body to the file. This supports huge files
	_, err = io.Copy(file, response.Body)
	if err != nil {
		return errors.Wrap(err, "LocalHTMLFromURL: failed to copy response body to file")
	}
	return nil
}

func GetURLWithRetry(url string, retries int) (*http.Response, error) {
	var err error
	var response *http.Response
	for i := 0; i <= retries; i++ {
		response, err = http.Get(url)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}
	}
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	return response, nil
}

func HTTPCookiesToSlice(cookies []*http.Cookie, list *[]string) {
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
			for i := 0; i < len(cookies)-2; i += 2 {
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

func waitvisible(timeoutSeconds int, sel string, opts ...chromedp.QueryOption) chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.WaitVisible(sel, opts...),
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
func waitFor(ctx context.Context, eventName string, timeout int) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		ch := make(chan string)
		cctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
		chromedp.ListenTarget(cctx, func(ev interface{}) {
			switch e := ev.(type) {
			case *page.EventLifecycleEvent:
				if e.Name == eventName {
					log.Infof("Received expected event: %s", eventName)
					cancel()
					ch <- eventName
					close(ch)
				}
			}
		})
		select {
		case event := <-ch:
			log.Infof("Done waiting for event %v", event)
			return nil
		case <-ctx.Done():
			log.Warn("Timeout waiting for event to occur")
			return ctx.Err()
		}
	}
}

// waitForJavaScript is an ActionFunc to wait for JavaScript to load.
func waitForJavaScript() chromedp.ActionFunc {
	return func(ctx context.Context) error {
		// Define a JS expression to check if document.readyState is "complete"
		jsExpression := `document.readyState === "complete";`

		// Evaluate the JS expression
		var jsResult bool
		err := chromedp.Run(ctx, chromedp.Evaluate(jsExpression, &jsResult))
		if err != nil {
			log.Errorf("Received error while waiting for javascript to complete: %v", err)
			return err
		}

		// If JS is not fully loaded, wait and poll until it is
		for !jsResult {
			// Sleep for a short duration before polling again
			time.Sleep(100 * time.Millisecond)

			// Re-evaluate the JavaScript expression
			err = chromedp.Run(ctx, chromedp.Evaluate(jsExpression, &jsResult))
			if err != nil {
				log.Errorf("Received error while waiting for javascript to complete: %v", err)
				return err
			}
		}
		log.Info("Document readyState is 'complete'")
		return nil
	}
}

func runWithTimeOut(ctx *context.Context, timeout time.Duration, tasks chromedp.Tasks) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		timeoutContext, cancel := context.WithTimeout(ctx, timeout*time.Second)
		defer cancel()
		return tasks.Do(timeoutContext)
	}
}

func URLScreenshotToPNG(url, imgFilePath string, sel interface{}, windowSize *[2]int, timeoutSeconds int, cookies ...string) error {
	opts := append(
		chromedp.DefaultExecAllocatorOptions[:],
		chromedp.UserAgent("Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/109.0.0.0 Safari/537.36"),
	)

	if windowSize != nil {
		log.Infof("Got windowSize with length 2, using window size of (%d, %d)", windowSize[0], windowSize[1])
		opts = append(opts, chromedp.WindowSize(windowSize[0], windowSize[1]))
	} else {
		opts = append(opts, chromedp.WindowSize(1920, 1200))
	}

	var timeout time.Duration
	if timeoutSeconds >= 15 {
		timeout = time.Duration(timeoutSeconds) * time.Second
	} else {
		timeout = time.Second * 15
	}

	ctxt := context.Background()
	// ctxt, cancel := context.WithTimeout(context.Background(), timeout)
	// defer cancel()

	allocCtx, allocCancel := chromedp.NewExecAllocator(ctxt, opts...)
	defer allocCancel()

	ctx, chromeCancel := chromedp.NewContext(allocCtx)
	defer chromeCancel()

	var err error
	var buf []byte
	var cookieTasks chromedp.Tasks
	cookieTasks = setcookies(url, cookies...)
	var selectorTask chromedp.QueryAction
	if sel != "" {
		selectorTask = chromedp.WaitVisible(sel)
		log.Infof("Created selector task with sel: %s", sel)
	}
	// capture entire browser viewport, returning png with quality=100
	// if len(cookies)%2 != 0 {
	// 	log.Warn("Length of cookies must be divisible by 2. Proceeding without setting cookies.")
	// 	if err = chromedp.Run(ctx,
	// 		// chromedp.Navigate(url),
	// 		chromedp.Sleep(time.Duration(timeout.Seconds()/2)),
	// 		// selectorTask,
	// 		chromedp.FullScreenshot(&buf, 100)); err != nil {
	// 		log.Fatal(err)
	// 		return err
	// 	}
	// } else {
	// log.Infof("Length of cookies is divisible by 2. Proceeding with selecting the proper Action Chain")
	if sel != "" {
		log.Infof("Selector was provided, proceeding with WaitVisible Action Chain")
		if err = chromedp.Run(ctx,
			cookieTasks,
			chromedp.Tasks{
				enableLifeCycleEvents(),
				waitFor(ctx, "networkIdle", timeoutSeconds/2),
			},
			// chromedp.Navigate(url),
			runWithTimeOut(&ctx, timeout, chromedp.Tasks{
				waitForJavaScript(),
				selectorTask,
			}),
			chromedp.FullScreenshot(&buf, 100)); err != nil {
			log.Fatal(err)
			return err
		}
	} else {
		log.Infof("Selector was not provided, proceeding with Sleep Action Chain of 5 seconds")
		if err = chromedp.Run(ctx,
			cookieTasks,
			// chromedp.Navigate(url),
			chromedp.Sleep(time.Duration(timeout.Seconds()/2)),
			runWithTimeOut(&ctx, timeout, chromedp.Tasks{
				waitForJavaScript(),
			}),
			chromedp.FullScreenshot(&buf, 100)); err != nil {
			log.Fatal(err)
			return err
		}
	}
	// }
	if f, err := os.Create(imgFilePath); err == nil {
		if _, err := f.Write(buf); err != nil {
			log.Fatal(err)
			return err
		}
	}
	return nil
}
