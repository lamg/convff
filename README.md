# `convff`

`convff` is a command line program for encapsulating several patterns for converting videos using `ffmpeg`:

## Usage

`convff` reads a list of video files, separated by newline character, from standard input and applies each one of them a conversion command selected according command line flags:

- `-d dir` where `dir` is the output directory for the converted files, it cannot be "." for avoiding name clashes.
- `-v` convert files for a VCD player
- `-x` convert files for a XVID player
- `-t` convert files for a digital TV receiver device, that reads MKV files with H264 video and Vorbis audio.
- `-w` convert files to WEBM format trying to copy streams compatible with WEBM.
- `-wo` convert files to WEBM with VP8 and Vorbis streams.
- `-wc` convert files to WEBM with VP9 and Opus streams.
- `-o` convert files to Opus.
- `-g` convert files to Vorbis with OGG as container.

## Core implementation

### Conversion arguments generator's interface

A conversion arguments generator for a specific video format returns a list of arguments for `ffmpeg`, according that format. Those arguments depend on information retrieved from the source file, like `convPar` or the output file name, which is equal to the input file name, but with the output file extension.

```go "conversion arguments generator interface"
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
```

### Converting several files

It's an advantage convert several files with one call, with that in mind the procedure `commands` is implemented:

```go "commands"
<<<conversion arguments generator interface>>>

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

<<<videoInfo>>>
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

### Command line interface

With the implementation of `commands` all that remains is supplying it paramaters, which come from the command line interface, and executing the commands it creates, which is done using `"os/exec"` library. The `fs` parameter is a list of file names parsed from standard input, this makes easy collecting the file names in a text file for then passing them in `convff` standard input.

The `main` procedure is implemented by:

```go "main content"
var dest string
var fvcd, fxvid, fdtv, w, wo, wc, o, g bool
flag.StringVar(&dest, "d", "", "Destination folder")
flag.BoolVar(&fvcd, "v", false, "VCD player target")
flag.BoolVar(&fxvid, "x", false, "XVID player target")
flag.BoolVar(&fdtv, "t", false, "Digital TV device target")
flag.BoolVar(&w, "w", false, "Faster conversion to WEBM")
flag.BoolVar(&wo, "wo", false, "WEBM with VP8 and Vorbis")
flag.BoolVar(&wo, "wc", false, "WEBM with VP9 and Opus")
flag.BoolVar(&o, "o", false, "Opus audio file")
flag.BoolVar(&g, "g", false, "Vorbis with OGG container file")
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

```go "read file names"
s, r := bufio.NewScanner(os.Stdin), make([]string, 0)
for s.Scan() {
	t := s.Text()
	r = append(r, t)
}
```

```go "command generation and running"
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
} else if o {
	cs = commands(r, dest, opus)
} else if g {
	cs = commands(r, dest, ogg)
}

inf := func(i int) {
	cs[i].Stdout, cs[i].Stderr = os.Stdout, os.Stderr
	cs[i].Run()
}
forall(inf, len(cs))
```

## Implementation of conversion argument generators

Notice that `vcd`, `xvid`, `mkv`, `fastWebm`, `oldWebm`, `currWebm` and `opus` are implementations of `convArgs`.

### DVD player as target

When the target is a DVD player, it accepts two formats: VCD and XVID. Conversion commands for both formats are generated respectively by:

```go "vcd"
type outExt func(string) string

func vcd(cp getCP, oe outExt) (args []string, e error) {
	out := oe(".mpg")
	args = []string{"-b:v", "1000k", "-b:a", "128k",
		"-target", "ntsc-dvd", out}
	return
}
```

```go "xvid"
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
```

### Digital TV receiver device as target

When the target is a digital TV receiver device the command is more complicated. Since it can reproduce several video and audio formats, using the `copy` feature there's opportunity to reduce the conversion time.

For video streams with H264 codec, tested with at most 50 frames per second (fps), there is no need to convert such streams. In that case the arguments for `ffmpeg` contain `-vcodec copy`. If the video stream is encoded with other format different from h264, it needs to be converted.

The digital TV device reproduces sound with MP3 and AAC codecs, therefore if the input audio stream has that format, the arguments to `ffmpeg` contain `-acodec copy`. Also that device reads MKV files, and that is the container selected in this case. This is implemented by:

```go "mkv"
func mkv(cp getCP, oe outExt) (args []string, e error) {
	n, e := cp()
	if e == nil {
		if n.audioC == "aac" || n.audioC == "mp3" || n.audioC == "vorbis" {
			n.audioC = "copy"
		} else {
			n.audioC = "libvorbis"
		}
		args = []string{"-acodec", n.audioC}
		if n.videoC == "h264" {
			n.videoC = "copy"
		} else {
			n.videoC = "h264"
		}
		out := oe(".mkv")
		args = append(args, "-vcodec", n.videoC, out)
	}
	return
}
```

### Converting to WEBM format

This is an MKV variant that contains VP8 or VP9 video streams, and Vorbis or Opus audio streams.

The `fastWebm` procedure tries to copy streams compatible with the WEBM format.

```go "fast webm"
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
```

The `oldWebm` procedure forces to a WEBM with VP8 and Vorbis.

```go "old webm"
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
```

The `currWebm` procedure forces to a WEBM with VP9 and Opus.

```go "current webm"
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
```

The WEBM conversion section includes:

```go "webm"
<<<fast webm>>>

<<<old webm>>>

<<<current webm>>>
```

### Converting to Opus and OGG

```go "opus and ogg"
func opus(cp getCP, oe outExt) (args []string, e error){
	n, e := cp()
	if e == nil {
		if n.audioC == "opus" {
			n.audioC = "copy"
		} else {
			n.audioC = "libopus"
		}
		args = []string{"-acodec", n.audioC}
		out := oe(".opus")
		args = append(args, out)
	}
	return
}

func ogg(cp getCP, oe outExt) (args []string, e error){
	n, e := cp()
	if e == nil {
		if n.audioC == "vorbis" {
			n.audioC = "copy"
		} else {
			n.audioC = "libvorbis"
		}
		args = []string{"-acodec", n.audioC}
		out := oe(".ogg")
		args = append(args, out)
	}
	return
}
```

### Source file

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

<<<webm>>>

<<<opus and ogg>>>
```

### TODO

`lmt` needs this for proper output, it must be an error

```go

```
