package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

type RivenInfo struct {
	name         string
	lowestPrice  int
	lowestPrice2 int
}

type Riven struct {
	Name string `json:"slug"`
}

type Response struct {
	Data []Riven `json:"data"`
}

func pullRivenList() []Riven {
	var response Response

	resp, err := http.Get("https://api.warframe.market/v2/riven/weapons")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		panic(err)
	}
	return response.Data
}

func pullRivenInfo(rivenSlugList []Riven) []RivenInfo {
	var rivenList []RivenInfo

	for _, item := range rivenSlugList {
		var riven RivenInfo
		url := "https://api.warframe.market/v1/auctions/search?type=riven&sort_by=price_asc&weapon_url_name=" + item.Name

		var resp *http.Response
		var err error
		for retry := 0; retry < 100; retry++ {
			resp, err = http.Get(url)
			if err == nil && resp.StatusCode == 200 {
				break
			}
			fmt.Println("retrying", item.Name, "attempt", retry+1)
			time.Sleep(time.Second * 5)
		}
		if err != nil {
			fmt.Println("skipping", item.Name, "error:", err)
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("read fail", item.Name)
			continue
		}

		var data map[string]any
		if err := json.Unmarshal(body, &data); err != nil {
			fmt.Println("bad JSON for", item.Name)
			continue
		}

		payload, ok := data["payload"].(map[string]any)
		if !ok {
			fmt.Println("no payload for", item.Name)
			continue
		}

		allAuctions, ok := payload["auctions"].([]any)
		if !ok {
			fmt.Println("no auctions field for", item.Name)
			continue
		}

		// filter only ingame
		var ingameAuctions []map[string]any
		for _, a := range allAuctions {
			auction, ok := a.(map[string]any)
			if !ok {
				continue
			}
			owner, ok := auction["owner"].(map[string]any)
			if !ok {
				continue
			}
			if status, ok := owner["status"].(string); ok && status == "ingame" {
				ingameAuctions = append(ingameAuctions, auction)
			}
		}

		if len(ingameAuctions) < 2 {
			fmt.Println("not enough ingame auctions for", item.Name)
			continue
		}

		price1 := ingameAuctions[0]["buyout_price"].(float64)
		price2 := ingameAuctions[1]["buyout_price"].(float64)

		riven.name = item.Name
		riven.lowestPrice = int(price1)
		riven.lowestPrice2 = int(price2)

		rivenList = append(rivenList, riven)
		fmt.Println("riven committed:", item.Name)

		time.Sleep(700 * time.Millisecond)
	}
	return rivenList
}


func main() {
	var endString string
	rivenSlugList := pullRivenList()
	rivenPriceList := pullRivenInfo(rivenSlugList)

	f, err := os.Create("./output.csv")
	if err != nil {
		panic(err)
	}

	endString = endString + fmt.Sprintln("name;price1;price2")

	for _, x := range rivenPriceList {
		endString = endString + fmt.Sprintln(x.name+";"+strconv.Itoa(x.lowestPrice)+";"+strconv.Itoa(x.lowestPrice2))
	}

	f.Write([]byte(endString))
	f.Sync()
}
