// convff reads from standard input a list of video files for
// converting them to the format specified as argument, using
// ffmpeg and ffprobe.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"encoding/json"
)

func main() {
	var dest string
	var dvd, dtv bool
	flag.StringVar(&dest, "d", "", "Destination folder")
	flag.BoolVar(&dvd, "v", false, "DVD player target")
	flag.BoolVar(&dtv, "t", false, "Digital TV device target")
	flag.Parse()
	var e error
	if dest != "." && dest != "" {
		e = os.MkdirAll(dest, os.ModeDir|os.ModePerm)
	} else {
		e = fmt.Errorf(`Output directory cannot be "%s"`, dest)
	}
	if e == nil {
		s, r := bufio.NewScanner(os.Stdin), make([]string, 0)
		for s.Scan() {
			t := s.Text()
			r = append(r, t)
		}
	
		var cs []*exec.Cmd
		if dvd {
			cs = commands(r, dest, mpg)
		}
		if dtv {
			cs = commands(r, dest, mkv)
		}
		
		inf := func(i int) {
			cs[i].Stdout, cs[i].Stderr = os.Stdout, os.Stderr
			cs[i].Run()
		}
		forall(inf, len(cs))
	}
	if e != nil {
		log.Fatal(e)
	}
}

type convCmd func(string, outExt) (*exec.Cmd, error)

func commands(fs []string, opath string, cc convCmd) (cs []*exec.Cmd) {
	cs = make([]*exec.Cmd, len(fs))
	inf := func(i int) {
		oe := output(fs[i], opath)
		var e error
		cs[i], e = cc(fs[i], oe)
		if e != nil {
			log.Print(e)
		}
	}
	forall(inf, len(fs))
	return
}

func output(inp, opath string) (oe outExt) {
	oe = func(ext string) (out string) {
		iext := path.Ext(inp)
		outf := inp[:len(inp)-len(iext)] + ext
		out = path.Join(opath, outf)
		return
	}
	return
}

type outExt func(string) string

func mpg(inp string, oe outExt) (c *exec.Cmd, e error) {
	out := oe(".mpg")
	c = exec.Command("ffmpeg", "-hide_banner", "-i", inp,
		"-b:v", "1000k", "-b:a", "128k",
		"-target", "ntsc-dvd", out,
	)
	return
}

func mkv(inp string, oe outExt) (c *exec.Cmd, e error) {
	args := []string{"-hide_banner", "-i", inp}
	var n *convPar
	n, e = videoInfo(inp)
	if e == nil {
		if n.audioC == "mp3" {
			n.audioC = "copy"
		} else {
			n.audioC = "mp3"
		}
		args = append(args, "-acodec", n.audioC)
		if n.videoC == "h264" && n.fps <= 30 {
			n.videoC = "copy"
		} else {
			n.videoC = "h264"
		}
		if n.fps > 30 {
			args = append(args, "-r", "30")
		}
		out := oe(".mkv")
		args = append(args, "-vcodec", n.videoC, out)
		c = exec.Command("ffmpeg", args...)
	}
	return
}

type ffInfo struct {
	Streams []stream `json: "streams"`
}

type stream struct {
	// audio and video fields
	Codec_Name   string `json: "codec_name"`
	Codec_Type   string `json: "codec_type"`
	R_Frame_Rate string `json: "r_frame_rate"`
}

type convPar struct {
	audioC string
	videoC string
	fps    int
}

func videoInfo(inp string) (n *convPar, e error) {
	ic := exec.Command("ffprobe", "-loglevel", "8",
		"-hide_banner", "-print_format", "json", "-show_streams",
		inp)
	var bs []byte
	bs, e = ic.Output()
	info := new(ffInfo)
	if e == nil {
		e = json.Unmarshal(bs, info)
	}
	if e == nil {
		n = new(convPar)
		inf := func(i int) {
			str := info.Streams[i]
			if str.Codec_Type == "video" {
				n.videoC = str.Codec_Name
				var num, den int
				fmt.Sscanf(str.R_Frame_Rate, "%d/%d", &num, &den)
				n.fps = num / den
			}
			if str.Codec_Type == "audio" {
				n.audioC = str.Codec_Name
			}
		}
		forall(inf, len(info.Streams))
	}
	return
}
