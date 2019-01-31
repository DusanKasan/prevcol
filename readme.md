# Prevcol

Computing colors and stuff...

## How to run

`go run main.go -concurrency=4 -outfile=output.csv input.txt`

## Description

The program in `main.go` reads a list of image URLs from file specified by its argument. Then it concurrently fetches the images on these URLs and for each one computes the three most dominant colors. The level of concurrency - the number of image fetching/processing goroutines - is set according to the `concurrency` flag, defaulting to 10. The computed values are then written to a CSV file in the format of `url, color1, color2, color3` and the colors are represented as 24bit hex values with leading `#` (e.g. `#ffffff`). The path to the output file is specified by the `outfile` flag, defaulting to `output.csv`. The progress of the program is also outputted to the command line. If any error is encountered during the program execution (unreachable url, invalid image formatting) the program is exited and the output file is to be considered incomplete.

## Constraints and trade-offs

- Supports only jpeg and png
- Every processed file is kept in memory, this can cause problems in systems with smaller memory, large images or high concurrency setting.
- If image does not contain at least 3 colors, the missing values are ommitted in the output (e.g. `http://a.b/c.jpg,#ffffff,,` for white image)
- The concurrent workers do both fetching of data over http and processing of the image. To be as generalized as possible this makes most sense. As we can not guess what systems the program will run on and/or what will be their bottleneck (networking/cpu/memory).
- There is 1 worker for 1 url/image, meaning no parallel image processing.

## Possible upgrade paths

- Support more image formats can be added easily
- Implement streaming image decoders/processors so that the whole image doesn't have to be kept in memory. This should be doable for most formats.
- Fine tune concurrency when the specs of the system the program will run on are known. This means parallelizing image processing on systems with multiple cores, working pretty much consecutively on single threaded systems (maybe just concurrent image fetching), and so on.
- If the specifications and storage allows it, consider preloading the images to local storage by a separate program. Then this program would only handle image processing which can be highly concurrent.