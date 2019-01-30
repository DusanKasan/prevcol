package main

import (
	"bufio"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
)

const contentTypeImageJpeg = "image/jpeg"
const contentTypeImagePng = "image/png"

type prevalentColors struct {
	url    *url.URL
	color1 *uint64
	color2 *uint64
	color3 *uint64
}

func main() {
	// inputs
	parallelism := flag.Int("parallelism", 10, "set the number of urls/images processed inFile parallel")
	outFilePath := flag.String("outfile", "output.csv", "the name/path of the output file")
	flag.Parse()

	inFilePath := flag.Arg(0)
	if inFilePath == "" {
		log.Panic("input file can not be empty")
	}

	// open the input file
	inFile, err := os.Open(inFilePath)
	if err != nil {
		log.Panicf("unable to open file: %s", inFilePath)
	}

	// open the output file
	outFile, err := os.OpenFile(*outFilePath, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		log.Panicf("unable to open file: %s", inFilePath)
	}

	// create the input and output channels
	urlsToProcessChan := make(chan *url.URL)
	prevalentColorsChan := make(chan *prevalentColors)
	wg := sync.WaitGroup{}
	wg.Add(1)

	// create parallel image readers according to parallelism input
	for i := 0; i < *parallelism; i ++ {
		go func() {
			for u := range urlsToProcessChan {
				i, err := readImageFromURL(u)
				if err != nil {
					log.Panic(err)
				}

				c1, c2, c3 := getThreeMostPrevalentColorsInImage(i)
				prevalentColorsChan <- &prevalentColors{
					url:    u,
					color1: c1,
					color2: c2,
					color3: c3,
				}
			}
		}()
	}

	// read the input file and process the urls in separate goroutine
	go func() {
		scanner := bufio.NewScanner(inFile)
		for scanner.Scan() {
			in := scanner.Text()
			u, err := url.Parse(in)
			if err != nil {
				log.Panicf("input is not an URL: %s", in)
			}
			urlsToProcessChan <- u
			wg.Add(1)
		}
		close(urlsToProcessChan)
		wg.Done()
	}()

	// write inFile a separate goroutine
	go func() {
		// TODO: Missing specification on what to do with 0, 1, 2 color images
		// representing as - for now
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
			out := fmt.Sprintf("%s, %s, %s, %s\n", p.url.String(), c1, c2, c3)
			fmt.Fprint(outFile, out)
			log.Print(out)
			wg.Done()
		}
	}()

	wg.Wait()
}

// only reads jpegs and pngs
func readImageFromURL(u *url.URL) (image.Image, error) {
	resp, err := http.Get(u.String())
	if err != nil {
		log.Panicf("unable to reach url: %s, err: %v", u, err)
	}

	switch contentType := resp.Header.Get("Content-Type"); contentType {
	case contentTypeImageJpeg:
		i, err := jpeg.Decode(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("unable to decode body from url: %s as jpeg, err: %v", u, err)
		}

		return i, nil
	case contentTypeImagePng:
		i, err := png.Decode(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("unable to decode body from url: %s as png, err: %v", u, err)
		}

		return i, nil
	default:
		return nil, fmt.Errorf("invalid content type: %s on url: %s", u, contentType)
	}
}

func getThreeMostPrevalentColorsInImage(i image.Image) (*uint64, *uint64, *uint64) {
	colors := map[uint64]uint64{}
	bounds := i.Bounds()
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			r, g, b, _ := i.At(x, y).RGBA()
			// revert the precomputation done by the library to represent the
			// colors as uint8, then do bit shifts to represent the color as int64.
			c := uint64(r)>>8<<16 + uint64(g)>>8<<8 + uint64(b)>>8
			colors[c] = colors[c] + 1
		}
	}

	// this can be optimized by computing the values as we iterate the loop above
	// will increase performance for colorful images
	var c1, c2, c3 *uint64
	var n1, n2, n3 uint64
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
