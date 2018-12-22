package main

import (
	"encoding/json"
	"os/exec"
)

type ffInfo struct {
	Streams []stream `json: "streams"`
}

type stream struct {
	// audio and video fields
	Index                int    `json: "index"`
	Codec_Name           string `json: "codec_name"`
	Codec_Long_Name      string `json: "codec_long_name"`
	Profile              string `json: "profile"`
	Codec_Type           string `json: "codec_type"`
	Codec_Time_Base      string `json: "codec_time_base"`
	Codec_Tag_String     string `json: "codec_tag_string"`
	Codec_Tag            string `json: "codec_tag"`
	Width                int    `json: "width"`
	Height               int    `json: "height"`
	Coded_Width          int    `json: "coded_width"`
	Coded_Height         int    `json: "coded_height"`
	Has_B_Frames         int    `json: "has_b_frames"`
	Sample_Aspect_Ratio  string `json: "sample_aspect_ratio"`
	Display_Aspect_Ratio string `json: "display_aspect_ratio"`
	Pix_Fmt              string `json: "pix_fmt"`
	Level                int    `json: "level"`
	Color_Range          string `json: "color_range"`
	Color_Space          string `json: "color_space"`
	Color_Transfer       string `json: "color_transfer"`
	Color_Primaries      string `json: "color_primaries"`
	Chroma_Location      string `json: "chroma_location"`
	Refs                 int    `json: "refs"`
	Is_Avc               string `json: "is_avc"`
	Nal_Length_Size      string `json: "nal_length_size"`
	R_Frame_Rate         string `json: "r_frame_rate"`
	Avg_Frame_Rate       string `json: "avg_frame_rate"`
	Time_Base            string `json: "time_base"`
	Start_Pts            int    `json: "start_pts"`
	Start_Time           string `json: "start_time"`
	Duration_Ts          int    `json: "duration_ts"`
	Duration             string `json: "duration"`
	Bit_Rate             string `json: "bit_rate"`
	Bits_Per_Raw_Sample  string `json: "bits_per_raw_sample"`
	Nb_Frames            string `json: "nb_frames"`
	disposition          `json: "disposition"`
	Tags                 tags `json: "tags"`

	// audio specific fields
	Sample_Fmt      string `json: "sample_fmt"`
	Sample_Rate     string `json: "sample_rate"`
	Channels        int    `json: "channels"`
	Channel_Layout  string `json: "channel_layout"`
	Bits_Per_Sample int    `json: "bits_per_sample"`
	Max_Bit_Rate    string `json: "max_bit_rate"`
}

type disposition struct {
	Default          int `json: "default"`
	Dub              int `json: "dub"`
	Original         int `json: "original"`
	Comment          int `json: "comment"`
	Lyrics           int `json: "lyrics"`
	Karaoke          int `json: "karaoke"`
	Forced           int `json: "forced"`
	Hearing_Impaired int `json: "hearing_impaired"`
	Visual_Impaired  int `json: "visual_impaired"`
	Clean_Effects    int `json: "clean_effects"`
	Attached_Pic     int `json: "attached_pic"`
	Timed_Thumbnails int `json: "timed_thumbnails"`
}

type tags struct {
	Language     string `json: "language"`
	Handler_Name string `json: "handler_name"`
}

func videoInfo(inp string) (audioc, videoc string, e error) {
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
		inf := func(i int) {
			str := info.Streams[i]
			if str.Codec_Type == "video" {
				videoc = str.Codec_Name
			}
			if str.Codec_Type == "audio" {
				audioc = str.Codec_Name
			}
		}
		forall(inf, len(info.Streams))
	}
	return
}
