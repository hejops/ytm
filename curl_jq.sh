query=$1
# query="hindemith violin lebhaft"
curl -sL 'https://music.youtube.com/youtubei/v1/search' -H 'Content-Type: application/json' --data-raw '{"context":{"client":{"clientName":"WEB_REMIX","clientVersion":"1.20240904.01.01"}},"query":"'"$query"'","params":"EgWKAQIIAWoSEAMQBBAJEA4QChAFEBEQEBAV"}' |
	jq --compact-output '
		.contents |
		.tabbedSearchResultsRenderer |
		.tabs[] |
		.tabRenderer |
		.content |
		.sectionListRenderer |
		.contents[] |
		.musicShelfRenderer |
		.contents[] |
		.musicResponsiveListItemRenderer |
		.flexColumns[] |
		.musicResponsiveListItemFlexColumnRenderer |
		.text |
		.runs[]'
