package sgtm

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"moul.io/sgtm/pkg/sgtmpb"
	"moul.io/u"
)

type Download struct {
	URL           string
	Path          string
	YoutubeDLFile string
	YoutubeDL     YoutubeDLOutput
}

func DownloadPost(post *sgtmpb.Post, force bool) (*Download, error) {
	download := Download{
		YoutubeDLFile: fmt.Sprintf("dl/%d.info.json", post.ID),
	}
	if post.Provider == sgtmpb.Provider_SoundCloud {
		download.URL = post.URL
	}
	if download.URL == "" {
		return nil, fmt.Errorf("unsupported post")
	}

	// wrap youtube-dl
	if force || !u.FileExists(download.YoutubeDLFile) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(
			ctx,
			"youtube-dl",
			"--write-info-json",
			"-o", fmt.Sprintf("dl/%d.%%(ext)s", post.ID),
			download.URL,
		)
		if err := cmd.Run(); err != nil {
			return nil, err
		}
	}

	// read manifest file
	{
		f, err := os.Open(download.YoutubeDLFile)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		b, _ := ioutil.ReadAll(f)
		if err := json.Unmarshal(b, &download.YoutubeDL); err != nil {
			return nil, err
		}
	}

	download.Path = fmt.Sprintf("dl/%d.%s", post.ID, download.YoutubeDL.Ext)
	return &download, nil
}

type ReadSeekerCloser interface {
	io.ReadSeeker
	io.ReadCloser
}

func StreamPost(ipfs *ipfsWrapper, post *sgtmpb.Post) (ReadSeekerCloser, error) {
	if post.Provider != sgtmpb.Provider_IPFS {
		return nil, fmt.Errorf("provider %q not supported", post.Provider.String())
	}
	return ipfs.cat(post.IPFSCID, post.SizeBytes), nil
}

func ExtractBPM(p string) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(
		ctx,
		"sonic-annotator",
		"-q",
		"-d", "vamp:qm-vamp-plugins:qm-tempotracker:tempo",
		"-w", "csv", "--csv-stdout",
		p,
	)
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	content := strings.TrimSpace(string(out))
	if content == "" {
		return 0, fmt.Errorf("bpm not extracted")
	}
	var bpmTotal, bpmCount float64
	r := csv.NewReader(strings.NewReader(content))
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, fmt.Errorf("failed to read csv: %w", err)
		}
		if len(record) < 3 {
			return 0, fmt.Errorf("invalid format: %q", record)
		}
		f, err := strconv.ParseFloat(record[2], 64)
		if err != nil {
			return 0, fmt.Errorf("not a float: %q", record)
		}
		bpmTotal += f
		bpmCount++
	}
	average := bpmTotal / bpmCount
	average = math.Round(average*100) / 100
	return average, nil
}

type YoutubeDLOutput struct {
	Extractor          string      `json:"extractor"`
	Protocol           string      `json:"protocol"`
	UploadDate         string      `json:"upload_date"`
	LikeCount          int         `json:"like_count"`
	Duration           float64     `json:"duration"`
	Fulltitle          string      `json:"fulltitle"`
	PlaylistIndex      interface{} `json:"playlist_index"`
	ViewCount          int         `json:"view_count"`
	Playlist           interface{} `json:"playlist"`
	Title              string      `json:"title"`
	Filename           string      `json:"_filename"`
	Abr                int         `json:"abr"`
	ID                 string      `json:"id"`
	CommentCount       int         `json:"comment_count"`
	UploaderURL        string      `json:"uploader_url"`
	Thumbnail          string      `json:"thumbnail"`
	WebpageURLBasename string      `json:"webpage_url_basename"`
	DisplayID          string      `json:"display_id"`
	Description        string      `json:"description"`
	Format             string      `json:"format"`
	Timestamp          int         `json:"timestamp"`
	Preference         interface{} `json:"preference"`
	Uploader           string      `json:"uploader"`
	Genre              string      `json:"genre"`
	FormatID           string      `json:"format_id"`
	UploaderID         string      `json:"uploader_id"`
	Thumbnails         []struct {
		URL        string `json:"url"`
		Width      int    `json:"width,omitempty"`
		Resolution string `json:"resolution,omitempty"`
		ID         string `json:"id"`
		Height     int    `json:"height,omitempty"`
		Preference int    `json:"preference,omitempty"`
	} `json:"thumbnails"`
	License      string `json:"license"`
	URL          string `json:"url"`
	ExtractorKey string `json:"extractor_key"`
	Vcodec       string `json:"vcodec"`
	HTTPHeaders  struct {
		AcceptCharset  string `json:"Accept-Charset"`
		AcceptLanguage string `json:"Accept-Language"`
		AcceptEncoding string `json:"Accept-Encoding"`
		Accept         string `json:"Accept"`
		UserAgent      string `json:"User-Agent"`
	} `json:"http_headers"`
	RepostCount int    `json:"repost_count"`
	Ext         string `json:"ext"`
	WebpageURL  string `json:"webpage_url"`
	Formats     []struct {
		Ext         string      `json:"ext"`
		Protocol    string      `json:"protocol"`
		Preference  interface{} `json:"preference"`
		Vcodec      string      `json:"vcodec"`
		Format      string      `json:"format"`
		URL         string      `json:"url"`
		FormatID    string      `json:"format_id"`
		HTTPHeaders struct {
			AcceptCharset  string `json:"Accept-Charset"`
			AcceptLanguage string `json:"Accept-Language"`
			AcceptEncoding string `json:"Accept-Encoding"`
			Accept         string `json:"Accept"`
			UserAgent      string `json:"User-Agent"`
		} `json:"http_headers"`
		Abr int `json:"abr"`
	} `json:"formats"`
}
