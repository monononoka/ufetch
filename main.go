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
	output := flag.String("o", "", "output file path (default: name from URL)")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: ufetch [-o FILE] <URL|app:query|hf:query>\n\n")
		fmt.Fprintln(flag.CommandLine.Output(), "Download files from HTTP(S), GitHub Releases (app:), or Hugging Face (hf:).")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	candidates, err := protocols.NewRouter(protocols.GitHub{}, protocols.HuggingFace{}).Resolve(ctx, flag.Arg(0))
	if err != nil {
		fail(err)
	}
	resolved, err := chooseDownload(candidates)
	if err != nil {
		fail(err)
	}
	destination := *output
	if destination == "" {
		destination = resolved.Filename
	}
	destination = filepath.Clean(destination)

	fmt.Printf("Downloading %s\n", destination)
	lastUpdate := time.Time{}
	progress := func(done, total int64) {
		if time.Since(lastUpdate) < 150*time.Millisecond && (total < 0 || done < total) {
			return
		}
		lastUpdate = time.Now()
		if total > 0 {
			pct := float64(done) / float64(total) * 100
			fmt.Fprintf(os.Stderr, "\r%s / %s (%.1f%%)", humanBytes(done), humanBytes(total), pct)
		} else {
			fmt.Fprintf(os.Stderr, "\r%s downloaded", humanBytes(done))
		}
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
			size = humanBytes(candidate.Size)
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

func humanBytes(n int64) string {
	if n < 0 {
		return "unknown"
	}
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	value := float64(n)
	units := []string{"KB", "MB", "GB", "TB"}
	for _, unitName := range units {
		value /= unit
		if value < unit || unitName == "TB" {
			return fmt.Sprintf("%.1f %s", value, unitName)
		}
	}
	return fmt.Sprintf("%d B", n)
}

func fail(err error) {
	message := strings.TrimSpace(err.Error())
	fmt.Fprintln(os.Stderr, "ufetch:", message)
	os.Exit(1)
}
