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
	var fvcd, fxvid, fdtv, w, wo, wc bool
	flag.StringVar(&dest, "d", "", "Destination folder")
	flag.BoolVar(&fvcd, "v", false, "VCD player target")
	flag.BoolVar(&fxvid, "x", false, "XVID player target")
	flag.BoolVar(&fdtv, "t", false, "Digital TV device target")
	flag.BoolVar(&w, "w", false, "Faster conversion to WEBM")
	flag.BoolVar(&wo, "wo", false, "WEBM with VP8 and Vorbis")
	flag.BoolVar(&wo, "wc", false, "WEBM with VP9 and Opus")
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
		if fvcd {
			cs = commands(r, dest, vcd)
		} else if fxvid {
			cs = commands(r, dest, xvid)
		} else if fdtv {
			cs = commands(r, dest, mkv)
		} else if w {
			cs = commands(r, dest, fastWebm)
		} else if wo {
			cs = commands(r, dest, oldWebm)
		} else if wc {
			cs = commands(r, dest, currWebm)
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

type convArgs func(getCP, outExt) ([]string, error)
type getCP func() (*convPar, error)

type convPar struct {
	audioC string
	videoC string
	fps    int
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

func commands(fs []string, opath string, 
	cc convArgs) (cs []*exec.Cmd) {
	cs = make([]*exec.Cmd, 0 ,len(fs))
	inf := func(i int) {
		oe := output(fs[i], opath)
		cp := func() (n *convPar, e error){
			n, e = videoInfo(fs[i])
			return
		}
		args, e := cc(cp, oe)
		if e == nil {
			args = append([]string{"-hide_banner", "-i", fs[i]}, 
				args...)
			c := exec.Command("ffmpeg", args...)
			cs = append(cs, c)
		} else {
			log.Print(e)
		}
	}
	forall(inf, len(fs))
	return
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

type outExt func(string) string

func vcd(cp getCP, oe outExt) (args []string, e error) {
	out := oe(".mpg")
	args = []string{"-b:v", "1000k", "-b:a", "128k",
		"-target", "ntsc-dvd", out}
	return
}

func xvid(cp getCP, oe outExt) (args []string, e error) {
	n, e := cp()
	if e == nil {
		args = []string{}
		if n.fps > 30 {
			args = append(args, "-r", "30")
		}
		if n.audioC == "ac3" {
			n.audioC = "copy"
		} else {
			n.audioC = "mp3"
		}
		out := oe(".mp4")
		args = append(args, "-b:v", "1000k",
			"-vcodec", "libxvid", "-s", "720x576",
			"-aspect:v", "4:3", "-acodec", n.audioC, out)
	}
	return
}

func mkv(cp getCP, oe outExt) (args []string, e error) {
	n, e := cp()
	if e == nil {
		if n.audioC == "mp3" {
			n.audioC = "copy"
		} else {
			n.audioC = "mp3"
		}
		args = []string{"-acodec", n.audioC}
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

func fastWebm(cp getCP, oe outExt) (args []string, e error) {
	n, e := cp()
	if e == nil {
		if n.audioC == "vorbis" || n.audioC == "opus" {
			n.audioC = "copy"
		} else {
			n.audioC = "libopus"
		}
		args = []string{"-acodec", n.audioC}
		if n.videoC == "vp9" || n.videoC == "vp8" {
			n.videoC = "copy"
		} else {
			n.videoC = "vp9"
		}
		out := oe(".webm")
		args = append(args, "-vcodec", n.videoC, out)
	}
	return
}

func oldWebm(cp getCP, oe outExt) (args []string, e error) {
	n, e := cp()
	if e == nil {
		if n.audioC == "vorbis" {
			n.audioC = "copy"
		} else {
			n.audioC = "libvorbis"
		}
		args = []string{"-acodec", n.audioC}
		if n.videoC == "vp8" {
			n.videoC = "copy"
		} else {
			n.videoC = "vp8"
		}
		out := oe(".webm")
		args = append(args, "-vcodec", n.videoC, out)
	}
	return
}

func currWebm(cp getCP, oe outExt) (args []string, e error) {
	n, e := cp()
	if e == nil {
		if n.audioC == "opus" {
			n.audioC = "copy"
		} else {
			n.audioC = "libopus"
		}
		args = []string{"-acodec", n.audioC}
		if n.videoC == "vp9" {
			n.videoC = "copy"
		} else {
			n.videoC = "vp9"
		}
		out := oe(".webm")
		args = append(args, "-vcodec", n.videoC, out)
	}
	return
}
