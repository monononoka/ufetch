package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/monononoka/ufetch/core"
	"github.com/monononoka/ufetch/protocols"
)

func main() {
	output := flag.String("o", "", "write output to FILE (default: file name from source)")
	directory := flag.String("dir", "", "save the selected file in DIR")
	flag.Usage = func() {
		out := flag.CommandLine.Output()
		fmt.Fprintln(out, "Usage:")
		fmt.Fprintln(out, "  ufetch [options] <URL|app:query|hf:query|npm:package|pypi:project>")
		fmt.Fprintln(out, "\nOptions:")
		flag.PrintDefaults()
		fmt.Fprintln(out, "\nProtocols:")
		fmt.Fprintln(out, "  URL       Direct HTTP or HTTPS download")
		fmt.Fprintln(out, "  app:      GitHub Releases (owner/repository or search terms)")
		fmt.Fprintln(out, "  hf:       Hugging Face model search or repository")
		fmt.Fprintln(out, "  npm:      Public npm package archive")
		fmt.Fprintln(out, "  pypi:     Public Python Package Index archive")
		fmt.Fprintln(out, "\nExamples:")
		fmt.Fprintln(out, "  ufetch https://example.com/file.zip")
		fmt.Fprintln(out, "  ufetch -dir ./downloads app:cli/cli")
		fmt.Fprintln(out, "  ufetch -o ./model.safetensors hf:org/model/path/to/model.safetensors")
		fmt.Fprintln(out, "  ufetch npm:lodash@4.17.21")
		fmt.Fprintln(out, "  ufetch pypi:requests")
	}
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	candidates, err := protocols.NewRouter(protocols.GitHub{}, protocols.HuggingFace{}, protocols.NPM{}, protocols.PyPI{}).Resolve(ctx, flag.Arg(0))
	if err != nil {
		fail(err)
	}
	resolved, err := chooseDownload(candidates)
	if err != nil {
		fail(err)
	}
	destination, err := resolveDestination(*output, *directory, resolved.Filename)
	if err != nil {
		fail(err)
	}

	fmt.Printf("Downloading to %s\n", destination)
	lastUpdate := time.Time{}
	progress := func(done, total int64) {
		if time.Since(lastUpdate) < 150*time.Millisecond && (total < 0 || done < total) {
			return
		}
		lastUpdate = time.Now()
		fmt.Fprintf(os.Stderr, "\r%s", core.FormatProgress(done, total))
	}
	if err := core.NewDownloader().Download(ctx, resolved.URL, destination, progress); err != nil {
		fmt.Fprintln(os.Stderr)
		if ctx.Err() != nil {
			fail(fmt.Errorf("download interrupted; partial data is kept for a later resume"))
		}
		fail(err)
	}
	fmt.Fprintln(os.Stderr)
	fmt.Printf("Saved to %s\n", destination)
}

func resolveDestination(output, directory, filename string) (string, error) {
	if output != "" && directory != "" {
		return "", fmt.Errorf("-o and -dir cannot be used together")
	}
	if output != "" {
		return filepath.Clean(output), nil
	}

	name := filepath.Base(filepath.FromSlash(filename))
	if name == "" || name == "." || name == ".." || name == string(filepath.Separator) {
		return "", fmt.Errorf("source did not provide a valid file name; specify one with -o")
	}
	if directory != "" {
		return filepath.Join(directory, name), nil
	}
	return filepath.Clean(name), nil
}

func chooseDownload(candidates []protocols.Download) (protocols.Download, error) {
	if len(candidates) == 0 {
		return protocols.Download{}, fmt.Errorf("the protocol returned no downloadable files")
	}
	if len(candidates) == 1 {
		return candidates[0], nil
	}

	fmt.Println("Available files:")
	for i, candidate := range candidates {
		label := candidate.Description
		if label == "" {
			label = candidate.Filename
		}
		size := "size unknown"
		if candidate.Size >= 0 {
			size = core.FormatBytes(candidate.Size)
		}
		fmt.Printf("  %d) %s (%s)\n", i+1, label, size)
	}
	fmt.Print("Select a file number: ")
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && strings.TrimSpace(line) == "" {
		return protocols.Download{}, fmt.Errorf("read selection: %w", err)
	}
	choice, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil || choice < 1 || choice > len(candidates) {
		return protocols.Download{}, fmt.Errorf("selection must be a number from 1 to %d", len(candidates))
	}
	return candidates[choice-1], nil
}

func fail(err error) {
	message := strings.TrimSpace(err.Error())
	fmt.Fprintln(os.Stderr, "ufetch:", message)
	os.Exit(1)
}
