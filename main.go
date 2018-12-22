package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
)

func main() {
	var dest string
	var tmpg, tmkv bool
	flag.StringVar(&dest, "d", "", "Destination folder")
	flag.BoolVar(&tmpg, "m", false, "Target MPG player")
	flag.BoolVar(&tmkv, "k", false, "Target MKV player")
	flag.Parse()
	r := make([]string, 0)
	var e error
	if dest != "." {
		s := bufio.NewScanner(os.Stdin)
		for s.Scan() {
			t := s.Text()
			r = append(r, t)
		}
	} else {
		e = fmt.Errorf("Output directory cannot be")
	}
	var cs []*exec.Cmd
	if e == nil {
		if tmpg {
			cs = commands(r, dest, mpg)
		}
		if tmkv {
			cs = commands(r, dest, mkv)
		}
	}
	inf := func(i int) {
		cs[i].Stdout, cs[i].Stderr = os.Stdout, os.Stderr
		cs[i].Run()
	}
	forall(inf, len(cs))
}

type outExt func(string) string

func mpg(inp string, oe outExt) (c *exec.Cmd, e error) {
	out := oe(".mpg")
	c = exec.Command("ffmpeg", "-hide_banner", "-i", inp, "-b:v", "1000k",
		"-b:a", "128k", "-target", "ntsc-dvd", out,
	)
	return
}

func mkv(inp string, oe outExt) (c *exec.Cmd, e error) {
	out := oe(".mkv")
	var ac, vc string
	ac, vc, e = videoInfo(inp)
	if ac == "mp3" {
		ac = "copy"
	} else {
		ac = "mp3"
	}
	if vc == "h264" {
		vc = "copy"
	} else {
		vc = "h264"
	}
	c = exec.Command("ffmpeg", "-hide_banner", "-i", inp, "-acodec", ac,
		"-vcodec", vc, out)
	return
}

func output(inp, oext, opath string) (out string) {
	ext := path.Ext(inp)
	outf := inp[:len(inp)-len(ext)] + oext
	out = path.Join(opath, outf)
	return
}

type convCmd func(string, outExt) (*exec.Cmd, error)

func commands(fs []string, opath string, cc convCmd) (cs []*exec.Cmd) {
	cs = make([]*exec.Cmd, len(fs))
	inf := func(i int) {
		oe := func(ext string) (p string) {
			p = output(fs[i], ext, opath)
			return
		}
		var e error
		cs[i], e = cc(fs[i], oe)
		if e != nil {
			log.Print(e)
		}
	}
	forall(inf, len(fs))
	return
}
