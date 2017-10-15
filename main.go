package main

import (
	"bufio"
	"flag"
	"os"
	"os/exec"
	"path"
)

func main() {
	var dest string
	flag.StringVar(&dest, "d", "", "Destination folder")
	flag.Parse()
	s, r := bufio.NewScanner(os.Stdin), make([]string, 0)
	for s.Scan() {
		r = append(r, s.Text())
	}

	for _, j := range r {

		cmd := exec.Command("ffmpeg",
			"-i", j,
			"-acodec", "libvorbis",
			"-vcodec", "copy",
			path.Join(dest, j),
		)
		cmd.Stderr, cmd.Stdout= os.Stderr, os.Stdout
		cmd.Run()
	}
}
