package main

import (
	"bufio"
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"net/http"
	"os"
	"sync"
)

type prevalentColors struct {
	url    string
	color1 *uint32
	color2 *uint32
	color3 *uint32
}

func main() {
	// parse inputs
	concurrency := flag.Int("concurrency", 10, "set the number of urls/images processed concurrently")
	outFilePath := flag.String("outfile", "output.csv", "the name/path of the output file")
	flag.Parse()
	inFilePath := flag.Arg(0)
	if inFilePath == "" {
		log.Fatalf("input file can not be empty")
	}

	// open the input file
	inFile, err := os.Open(inFilePath)
	if err != nil {
		log.Fatalf("unable to open input file: %s", inFilePath)
	}

	// open the output file
	outFile, err := os.OpenFile(*outFilePath, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		log.Fatalf("unable to open output file: %s", inFilePath)
	}

	// create the input/output channels and wait group
	urlsToProcessChan := make(chan string)
	prevalentColorsChan := make(chan *prevalentColors)
	wg := sync.WaitGroup{}

	// create parallel image readers according to concurrency input
	for i := 0; i < *concurrency; i ++ {
		go func() {
			for url := range urlsToProcessChan {
				i, err := readImageFromURL(url)
				if err != nil {
					log.Fatalf("unable to read image from url: %s, err: %v", url, err)
				}

				c1, c2, c3 := getThreeMostPrevalentColorsInImage(i)
				prevalentColorsChan <- &prevalentColors{
					url:    url,
					color1: c1,
					color2: c2,
					color3: c3,
				}
			}
		}()
	}

	// write the output file in a separate goroutine
	go func() {
		for p := range prevalentColorsChan {
			var c1, c2, c3 string
			if p.color1 != nil {
				c1 = fmt.Sprintf("#%06x", *p.color1)
			}
			if p.color2 != nil {
				c2 = fmt.Sprintf("#%06x", *p.color2)
			}
			if p.color3 != nil {
				c3 = fmt.Sprintf("#%06x", *p.color3)
			}
			out := fmt.Sprintf("%s, %s, %s, %s\n", p.url, c1, c2, c3)
			fmt.Fprint(outFile, out)
			log.Print(out)
			wg.Done()
		}
	}()

	// read the input file and process the urls in separate goroutine
	scanner := bufio.NewScanner(inFile)
	for scanner.Scan() {
		urlsToProcessChan <- scanner.Text()
		wg.Add(1)
	}

	wg.Wait()
}

// readImageFromURL returns the image from a provided url. If the url does not
// contain image, it can not be fetched or parsed it returned an error.
func readImageFromURL(url string) (image.Image, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("unable to reach url: %s, err: %v", url, err)
	}

	i, _, err := image.Decode(resp.Body)
	resp.Body.Close()
	return i, err
}

// getThreeMostPrevalentColorsInImage iterates over each pixel in the image to
// determine the three most prevalent colors. Since the output should range from
// #000000 to #ffffff we represent primary colors as uint8 which means in some
// cases this can be lossy.
func getThreeMostPrevalentColorsInImage(i image.Image) (*uint32, *uint32, *uint32) {
	var c1, c2, c3 *uint32
	var n1, n2, n3 uint64

	// map of color to number of its occurrences in the image
	bounds := i.Bounds()
	colors := make(map[uint32]uint64, bounds.Dx()*bounds.Dy())
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			r, g, b, _ := i.At(x, y).RGBA()
			// revert the precomputation done by the library to represent the
			// primary colors as uint8, then do bit shifts to represent the
			// resulting color as uint32. This is done here manually to avoid
			// the overhead of calling `color.RGBAModel` and casting to `color.RGBA`.
			c := r>>8<<16 + g>>8<<8 + b>>8
			colors[c] = colors[c] + 1
		}
	}

	for c, n := range colors {
		cc := c
		switch {
		case n > n1:
			c3 = c2
			n3 = n2
			c2 = c1
			n2 = n1
			c1 = &cc
			n1 = n
		case n > n2:
			c3 = c2
			n3 = n2
			c2 = &cc
			n2 = n
		case n > n3:
			c3 = &cc
			n3 = n
		}
	}

	return c1, c2, c3
}
