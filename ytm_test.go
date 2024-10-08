package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJSONResponse(t *testing.T) {
	query := "death grips"
	// "rousset duphly"
	// "hwv 427 allegro"

	// var out1 any
	// if err := json.Unmarshal(searchGo(query), &out1); err != nil {
	// 	panic(err)
	// }
	// assert.Equal(t, strings.Count(spew.Sdump(out1), "videoId"), 91)
	//
	// var out2 any
	// if err := json.Unmarshal(searchCurl(query), &out2); err != nil {
	// 	panic(err)
	// }
	// assert.Equal(t, strings.Count(spew.Sdump(out2), "videoId"), 160)

	// assert.Equal(t, strings.Count(string(searchCurlJq(query)), "videoId"), 20)
	// assert.Equal(t, len(parseCurlJq(searchCurlJq(query))), 20)

	b := searchCurlJq(query)
	assert.Equal(t, strings.Count(string(b), "videoId"), len(parseCurlJq(b)))

	assert.Equal(
		t,
		(&Album{Id: "MPREb_BL9sWaZWAUE"}).getPlaylistId(),
		"OLAK5uy_lvmV9P8LsaV83ALc6PrdRleHAtSwKzkHQ", // pragma: allowlist secret
	)
	assert.Equal(
		t,
		(&Album{Id: "MPREb_6KTedIfvZMt"}).getPlaylistId(),
		"OLAK5uy_nMLv2tlhhUVZHZQ5PQ7hzp1n9w-OqcQ44", // pragma: allowlist secret
	)

	b = searchCurlJq("hindemith lebhaft violin")
	assert.Len(t, parseCurlJq(b)[19].Id, 11)
}
