# `convff`

`convff` is a command line program for encapsulating two patterns for converting videos using `ffmpeg`:

## DVD player as target

When the target is a DVD player, it accepts two formats: VCD and XVID. Conversion commands for both formats are generated respectively by:

```go "vcd"
type outExt func(string) string

func vcd(inp string, oe outExt) (c *exec.Cmd, e error) {
	out := oe(".mpg")
	c = exec.Command("ffmpeg", "-hide_banner", "-i", inp,
		"-b:v", "1000k", "-b:a", "128k",
		"-target", "ntsc-dvd", out,
	)
	return
}
```

```go "xvid"
func xvid(inp string, oe outExt) (c *exec.Cmd, e error) {
	out := oe(".avi")
	c = exec.Command("ffmpeg", "-hide_banner", "-i", inp,
		"-b:v", "700k", "-vcodec", "libxvid", "-s", "720x576",
		"-r", "30", "-aspect:v", "4:3", "-acodec", "mp3", out)
	return
}
```

The `outExt` implementation carries the output directory joined with the output file name in its context, when passing the file extension as parameter it returns the complete output path.

## Digital TV device as target

When the target is a digital TV device the command is more complicated. Since it can reproduce several video and audio formats, using the `copy` feature there's opportunity to reduce the conversion time.

For video streams with h264 codec at 30 frames per second (fps) and less, there is no need to convert such streams. In that case the arguments for `ffmpeg` contain `-vcodec copy`. If the video stream is encoded with other format different from h264, it needs to be converted. Also, regardless of the video codec, if `fps > 30` the arguments to `ffmpeg` contain `-vcodec h264 -r 30`.

The digital TV device reproduces mp3 sound, therefore if the input audio stream has that format, the arguments to `ffmpeg` contain `-acodec copy`. Also that device reads MKV files, and that is the container selected in this case. This is implemented by:

```go "mkv"
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
```

This procedure depends on `videoInfo`, which given an input file returns information with the video and audio codecs in that file, also with the fps number. This is implemented leveraging on `ffprobe` a command accompanying `ffmpeg`, for getting information about media files. It can output information in JSON format, which is an advantage since Go already has a JSON decoder in the standard library. This decoder needs a Go data type for decoding the JSON text, it is:

```go "ffinfo"
type ffInfo struct {
	Streams []stream `json: "streams"`
}

type stream struct {
	// audio and video fields
	Codec_Name   string `json: "codec_name"`
	Codec_Type   string `json: "codec_type"`
	R_Frame_Rate string `json: "r_frame_rate"`
}
```

The `ffprobe` output has more fields, but they are irrelevant in this case, and if they have no homologue in the Go data type, the decoder can ignore them while parsing those defined. Assuming the input streams has only one video stream, and only one audio stream, and that the order of them isn't known, the implementation is:

```go "videoInfo"
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
```

Scanning `num` and `den`, and then doing `n.fps = num / den` is necessary since some times the `R_Frame_Rate` field comes in the form `"30000/1001"`.

## Converting several files

It's an advantage convert several files with one call, with that in mind the procedure `commands` is implemented:

```go "commands"
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
```

Notice that both `mpg` and `mkv` are implementations of `convCmd`. The `output` procedure returns a procedure that creates the output file path according information captured in this context (`inp`, `opath`), and an output extension supplied in the context where its called.

## Command line interface

With the implementation of `commands` all that remains is supplying it paramaters, which come from the command line interface, and executing the commands it creates, which is done using `"os/exec"` library. The `fs` parameter is a list of file names parsed from standard input, this makes easy collecting the file names in a text file for then passing them in `convff` standard input.

```go "read file names"
s, r := bufio.NewScanner(os.Stdin), make([]string, 0)
for s.Scan() {
	t := s.Text()
	r = append(r, t)
}
```

With command line flag `dvd` (DVD player) and `dtv` (digital TV device), generating and running commands is done by:

```go "command generation and running"
var cs []*exec.Cmd
if fvcd {
	cs = commands(r, dest, vcd)
}
if fxvid {
	cs = commands(r, dest, xvid)
}
if fdtv {
	cs = commands(r, dest, mkv)
}

inf := func(i int) {
	cs[i].Stdout, cs[i].Stderr = os.Stdout, os.Stderr
	cs[i].Run()
}
forall(inf, len(cs))
```

`dest` is the command line argument defining the output directory.

The `main` procedure is implemented by:

```go "main content"
var dest string
var fvcd, fxvid, fdtv bool
flag.StringVar(&dest, "d", "", "Destination folder")
flag.BoolVar(&fvcd, "v", false, "VCD player target")
flag.BoolVar(&fxvid, "x", false, "XVID player target")
flag.BoolVar(&fdtv, "t", false, "Digital TV device target")
flag.Parse()
var e error
if dest != "." && dest != "" {
	e = os.MkdirAll(dest, os.ModeDir|os.ModePerm)
} else {
	e = fmt.Errorf(`Output directory cannot be "%s"`, dest)
}
if e == nil {
	<<<read file names>>>

	<<<command generation and running>>>
}
if e != nil {
	log.Fatal(e)
}
```

## Source file

The source file for this program is:

```go main.go
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
	<<<main content>>>
}

<<<commands>>>

<<<vcd>>>

<<<xvid>>>

<<<mkv>>>

<<<ffinfo>>>

<<<videoInfo>>>
```

## TODO

`lmt` needs this for proper output, it must be an error

```go

```