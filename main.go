package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
)

func main() {
	var dest, targ, dext, ac, vc, conv string
	flag.StringVar(&dest, "d", "", "Destination folder")
	flag.StringVar(&targ, "t", "", "Target extension")
	flag.StringVar(&dext, "e", "", "Output extension")
	flag.StringVar(&ac, "a", "", "Audio codec")
	flag.StringVar(&vc, "v", "", "Video codec")
	flag.StringVar(&conv, "c", "avconv", "Converter")
	flag.Parse()
	if dest != "." {
		s, r := bufio.NewScanner(os.Stdin), make([]string, 0)
		for s.Scan() {
			r = append(r, s.Text())
		}
		for _, j := range r {
			ext := path.Ext(j)
			if ext == targ {
				fl := j[:len(j)-len(ext)] + dext
				var cmd *exec.Cmd
				if ac != "" && vc != "" {
					cmd = exec.Command(conv,
						"-i", j,
						"-acodec", ac,
						"-vcodec", vc,
						path.Join(dest, fl),
					)
				} else if ac != "" {
					cmd = exec.Command(conv,
						"-i", j,
						"-acodec", ac,
						path.Join(dest, fl),
					)
				}
				if cmd != nil {
					cmd.Stderr, cmd.Stdout = os.Stderr, os.Stdout
					cmd.Run()
				}
			}
		}
	} else {
		fmt.Fprintf(os.Stderr, "Output directory cannot be .\n")
	}
}
