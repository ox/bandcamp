package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

// DataBlob is a rendered JSON blob that's returned on the initial page fetch. Subsequent fetches
// for data are done using POST requests to their API; urls look like
// https://bandcamp.com/api/fancollection/1/wishlist_items with the fan_id, and older_than_token.
// older_than_token refers to the last "last_token" that was given to the client. If you're going to
// be paginating the wishlist, you should use the "last_token" in the Wishlist struct.
type DataBlob struct {
	// TrackList is a list of tracks in your collection. The page blob contains roughly 40 or so
	// tracks.
	TrackList []BlobTrack `json:"track_list"`

	// ItemCache stores maps of id->item. Sequences use the ids and they should be fetched out of
	// these ItemCaches.
	ItemCache struct {
		Collection map[string]Item `json:"collection"`
		Wishlist   map[string]Item `json:"wishlist"`
	} `json:"item_cache"`

	CollectionData ItemData `json:"collection_data"`
	WishlistData   ItemData `json:"wishlist_data"`

	FanData struct {
		ID int `json:"fan_id"`
	} `json:"fan_data"`
}

// ItemData describes the items and their order for a particular kind of item. Item IDs are listed
// in `Sequence` and `PendingSequence`. Data for each item can be found by matching the Item ID to
// an ID in the appropriate `ItemCache`.
type ItemData struct {
	// LastToken is used as a pagination cursor to fetch the next batch of items
	LastToken string `json:"last_token"`

	// Sequence is the order of the items on the current page to be rendered in. See
	// `PendingSequence` for an explanation of what these lists mean.
	Sequence []string `json:"sequence"`

	// PendingSequence is a sequence that needs to be shown first. This is the first batch of items
	// for a given item category ("followers", "wishlist", etc) that isn't baked into the current
	// page.
	//
	// Visiting the wishlist page will result in an empted "pending_sequence" list, and items in the
	// "sequence" list, whereas "colleciton_data" will have a full "pending_sequence" list and an
	// empty "sequence" list.
	PendingSequence []string `json:"pending_sequence"`
}

type BlobTrack struct {
	BandName string `json:"band_namp"`
	Title    string `json:"title"`
	AlbumID  int    `json:"album_id"`
}

type APIItemsResponse struct {
	Items         []Item `json:"track_list"`
	MoreAvailable bool   `json:"more_available"`
	LastToken     string `json:"last_token"`
}

// Item represents an API return item. Ambiguous while exploring API. "purchased" will always be
// false for items in the wishlist
type Item struct {
	// Date added to wishlist
	Added string `json:"added"`

	// URL of the item in the wishlist
	ItemURL string `json:"item_url"`

	// Type of the item. Could be "album", or "track" (maybe "merch")
	ItemType string `json:"item_type"`
}

func GetWishlist(fanID, lastpageToken string) (APIItemsResponse, error) {
	request := map[string]string{
		"fan_id":           fanID,
		"older_than_token": lastpageToken,
	}
	requestJSON, _ := json.Marshal(request)

	resp, err := http.Post("https://bandcamp.com/api/fancollection/1/wishlist_items", "application/json", strings.NewReader(string(requestJSON)))
	if err != nil {
		return APIItemsResponse{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return APIItemsResponse{}, err
	}

	var response APIItemsResponse
	if err = json.Unmarshal(body, &response); err != nil {
		return APIItemsResponse{}, err
	}

	return response, nil
}

func main() {
	resp, err := http.Get("https://bandcamp.com/space-llama/wishlist")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Extract the baked-in datablob in the HTML
	datablobExp := regexp.MustCompile("id=\"pagedata\".*?data-blob=\"(.*?)\">")
	datablobMatch := datablobExp.FindStringSubmatch(string(body))
	pagedata := strings.ReplaceAll(datablobMatch[1], "&quot;", "\"")

	// Unmarshal the datablob
	var datablob DataBlob
	if err := json.Unmarshal([]byte(pagedata), &datablob); err != nil {
		fmt.Println(err)
		return
	}

	for _, trackID := range datablob.WishlistData.Sequence {
		for itemcacheID, item := range datablob.ItemCache.Wishlist {
			if trackID == itemcacheID {
				fmt.Println(item.ItemURL)
				break
			}
		}
	}

	nextPage, err := GetWishlist(strconv.Itoa(datablob.FanData.ID), datablob.WishlistData.LastToken)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, item := range nextPage.Items {
		fmt.Println(item.ItemURL)
	}
}
