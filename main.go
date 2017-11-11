package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
)

const mkv = ".mkv"

func main() {
	var dest string
	flag.StringVar(&dest, "d", "", "Destination folder")
	flag.Parse()
	if dest != "." {
		s, r := bufio.NewScanner(os.Stdin), make([]string, 0)
		for s.Scan() {
			r = append(r, s.Text())
		}
		acodec := "libvorbis"
		for _, j := range r {
			vcodec := "copy"
			fl, ext := j, path.Ext(j)
			if ext != mkv {
				fl, vcodec = j[:len(j)-len(ext)]+mkv, "h264"
			}
			cmd := exec.Command("ffmpeg",
				"-i", j,
				"-acodec", acodec,
				"-vcodec", vcodec,
				path.Join(dest, fl),
			)
			cmd.Stderr, cmd.Stdout = os.Stderr, os.Stdout
			cmd.Run()
		}
	} else {
		fmt.Fprintf(os.Stderr, "Output directory cannot be .\n")
	}
}
