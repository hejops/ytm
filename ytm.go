package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os/exec"
	"strings"
)

// 1. pure Go (fetch and parse) -- incomplete (?) response
//
// 2. fetch with curl, parse with Go -- full response, but traversing the tree
// is inane
//
// 3. fetch with curl, destructure with jq, parse with Go -- relatively
// straightforward, despite jsonl

const JQ_SCRIPT = "./curl_jq.sh" // TODO: can we use go:embed or something?

type YoutubeVideo struct {
	Title    string
	Id       string
	Artist   Artist
	Album    Album
	Duration string
	Plays    string // human-formatted (e.g. 10K)
}

type Run struct {
	Text               string
	NavigationEndpoint struct {
		WatchEndpoint  struct{ VideoId string }  // track
		BrowseEndpoint struct{ BrowseId string } // artist / album
	}
}

// {{{

var YT_PAYLOAD = map[string]any{
	"context": map[string]any{
		// https://github.com/zerodytrash/YouTube-Internal-Clients/blob/main/results/working_clients.txt
		// may be brittle; what does yt-dlp use?
		"client": map[string]string{
			"clientName":    "WEB_REMIX",
			"clientVersion": "1.20240904.01.01",
		},
		// // important to get more results
		// "params": "EgWKAQIIAWoSEAMQBBAJEA4QChAFEBEQEBAV",
		// "user": map[string]interface{}{
		// 	"lockedSafetyMode": false,
		// },
	},
}

// type YoutubeSchema struct {
// 	Contents struct {
// 		TabbedSearchResultsRenderer struct {
// 			Tabs []struct {
// 				TabRenderer struct {
// 					Content struct {
// 						SectionListRenderer struct {
// 							Contents []struct {
// 								MusicShelfRenderer struct {
// 									Contents []struct {
// 										MusicResponsiveListItemRenderer struct {
// 											FlexColumns []struct {
// 												MusicResponsiveListItemFlexColumnRenderer struct{ Text struct{ Runs []Run } }
// 											}
// 										}
// 									}
// 								}
// 							}
// 						}
// 					}
// 				}
// 			}
// 		}
// 	}
// }

// // pure Go implementation, only retrieves about 12 unique video ids
// func searchGo(query string) []byte {
// 	YT_PAYLOAD["query"] = query
// 	b, err := json.Marshal(YT_PAYLOAD)
// 	if err != nil {
// 		panic(err)
// 	}
// 	req, err := http.NewRequest(
// 		"POST",
// 		// "https://music.youtube.com/youtubei/v1/search?"+params.Encode(),
// 		"https://music.youtube.com/youtubei/v1/search",
// 		bytes.NewBuffer(b),
// 	)
// 	if err != nil {
// 		panic(err)
// 	}
// 	resp, err := http.DefaultClient.Do(req)
// 	if err != nil {
// 		panic(err)
// 	}
// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer resp.Body.Close()
// 	return body
// }

// // Retrieves 20 unique videos
// func searchCurl(query string) []byte {
// 	b, err := exec.Command(
// 		"curl",
// 		"-sL",
// 		"-H",
// 		"Content-Type: application/json",
// 		"--data",
// 		`{"context":{"client":{"clientName":"WEB_REMIX","clientVersion":"1.20240904.01.01"}},"query":"`+query+`","params":"EgWKAQIIAWoSEAMQBBAJEA4QChAFEBEQEBAV"}`,
// 		"https://music.youtube.com/youtubei/v1/search",
// 	).Output()
// 	if err != nil {
// 		panic(err)
// 	}
// 	return b
// }

// func parseCurl(b []byte) (any, error) {
// 	var out YoutubeSchema
//
// 	if err := json.Unmarshal(b, &out); err != nil {
// 		return nil, errors.New("failed to unmarshal")
// 	}
//
// 	// for _, tab := range out.Contents.TabbedSearchResultsRenderer.Tabs {
// 	// 	for _, c := range tab.TabRenderer.Content.SectionListRenderer.Contents {
// 	// 		c.MusicShelfRenderer.Contents
// 	// 	}
// 	// }
//
// 	return out, nil
// }

// }}}

func searchCurlJq(query string) []byte {
	b, err := exec.Command("bash", JQ_SCRIPT, query).Output()
	if err != nil {
		panic(err)
	}
	return b
}

type Artist struct {
	Name string
	Id   string
}

type Album struct {
	Name string
	Id   string
}

func parseCurlJq(b []byte) (videos []YoutubeVideo) {
	d := json.NewDecoder(bytes.NewBuffer(b))
	var v YoutubeVideo
	for i := 0; ; i++ {
		var line Run
		err := d.Decode(&line)
		if err == io.EOF {
			break // done decoding file
		} else if err != nil {
			panic(err)
		}

		// Generally, we expect 140 json lines, in the following order:
		// 0. song
		// 1. artist
		// 2. dot
		// 3. album
		// 4. dot
		// 5. duration
		// 6. plays

		t := line.Text
		switch i % 7 {
		case 0:
			v.Title = t
			v.Id = line.NavigationEndpoint.WatchEndpoint.VideoId
		case 1:
			v.Artist = Artist{Name: t, Id: line.NavigationEndpoint.BrowseEndpoint.BrowseId}
		case 3:
			v.Album = Album{Name: t, Id: line.NavigationEndpoint.BrowseEndpoint.BrowseId}
		case 5:
			v.Duration = t
		case 6:
			// n := strings.Fields(line.Text)[0]
			// x, err := strconv.Atoi(n)
			// if err != nil {
			// 	panic(err)
			// }
			v.Plays = strings.Fields(t)[0]
			videos = append(videos, v)
			v = YoutubeVideo{}

		case 2, 4:
			continue
		}

	}

	return videos
}

func (a *Album) getPlaylistId() string {
	// curl -sL 'https://music.youtube.com/youtubei/v1/browse?prettyPrint=false' -X POST -H 'Content-Type: application/json' --data-raw '{"context":{"client":{"clientName":"WEB_REMIX","clientVersion":"1.20240904.01.01"}},"browseId":"MPREb_BL9sWaZWAUE"}' | jq .

	if a.Id == "" {
		panic(1)
	}

	YT_PAYLOAD["browseId"] = a.Id
	b, err := json.Marshal(YT_PAYLOAD)
	if err != nil {
		panic(err)
	}
	req, err := http.NewRequest(
		"POST",
		"https://music.youtube.com/youtubei/v1/browse", // browse, not search!
		bytes.NewBuffer(b),
	)
	if err != nil {
		panic(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	s := string(body)
	i := strings.Index(s, "OLAK5uy")
	return s[i : i+41]
}
