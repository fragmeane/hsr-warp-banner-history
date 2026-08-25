package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"maps"
	"net/http"
	"os"
	"slices"
	"strconv"
	"sync"
)

// Scheduled task to run twice a day every day to monitor starrailstation API and StarRailRes updates.
// Performs necessary updates to the index when updates are found.
func monitor() {
	index, err := readIndex()

	if err != nil {
		log.Fatalln("Unable to read " + INDEXFILE + ". ERROR: " + err.Error())
		return
	}
	
	resp, err := http.Get(BANNERURL)

	if err != nil {
		log.Fatalln("Unable to get a response from starrailstation API. ERROR: " + err.Error())
		return
	}

	defer resp.Body.Close()
		
	lcResp, err := http.Get(LCURL)

	if err != nil {
		log.Fatalln("Unable to request light_cones.json from StarRailRes repository")
		return
	}

	defer lcResp.Body.Close()
	
	var response Response
	var lcResponse map[string]LightCone
	
	decoder := json.NewDecoder(resp.Body)
	decoder.Decode(&response)

	b, err := io.ReadAll(lcResp.Body)

	if err != nil {
		log.Fatalln("Unable to read StarRailRes light_cones.json content")
		return
	}

	json.Unmarshal(b, &lcResponse)

	charIndex, _ := strconv.Atoi(getLatestBanner(&index, CHARACTERGACHA).Id)
	lcIndex, _ := strconv.Atoi(getLatestBanner(&index, LCGACHA).Id)
	collabCharIndex, _ := strconv.Atoi(getLatestBanner(&index, COLLABCHARACTERGACHA).Id)
	collabLcIndex, _ := strconv.Atoi(getLatestBanner(&index, COLLABLCGACHA).Id)

	indices := []int{charIndex, lcIndex, collabCharIndex, collabLcIndex}

	changed := false

	for _, idx := range indices {
		count := 1
		gtype := CHARACTERGACHA

		if idx > STELLARBOUNDARY && idx < CHARACTERBOUNDARY {
			gtype = CHARACTERGACHA
		} else if idx < LCBOUNDARY {
			gtype = LCGACHA
		} else if idx > COLLABCHARBOUNDARY && idx < COLLABCHARBOUNDARY {
			gtype = COLLABCHARACTERGACHA
		}

		gachaType := strconv.Itoa(gtype)

		in:
		for {
			latest, exists := response.Config.Banners[idx + count]

			if !exists {
				break in
			}

			banner := Banner{
				Id : strconv.Itoa(idx + count),
				RateUp4: latest.RateUp4,
				RateUp: latest.RateUp,
				Rerun: latest.Rerun,
				GachaType: gachaType,
				CompanionBanner: latest.CompanionBanner,
				StartTime: latest.StartTime,
				EndTime: latest.EndTime,
				IsExclusive: true,
			}

			if banner.GachaType == strconv.Itoa(LCGACHA) {
				banner.Desc = lcResponse[strconv.Itoa(banner.RateUp)].Name
			}
			
			err = addBanner(&index, banner)

			if err == nil {
				changed = true
			}
			
			count++
		}
	}

	if changed {
		content, err := json.MarshalIndent(index, "", "    ")
		miniContent, err := json.Marshal(index)
	
		if err != nil {
			log.Fatalln("Unable to marshal content")
			return
		}
	
		err = writeToIndex(content, miniContent)
	
		if err != nil {
			log.Fatalln("Unable to write to " + INDEXFILE)
			return
		}

		log.Println("Successfully updated " + INDEXFILE)
	}
}


// One-time function for generating the index straight from the sources
func setup() {
	var wg sync.WaitGroup

	index := BannerHistory{Banners: make([]Banner, 0)}

	index, err := readIndex()

	if err != nil {
		log.Fatalln("Unable to read index.json. ERROR: " + err.Error())
		return
	}

	wg.Go(func() {
		var response Response

		resp, err := http.Get(BANNERURL)

		if err != nil {
			log.Fatalln("Unable to get a response from starrailstation API. ERROR: " + err.Error())
			return
		}

		defer resp.Body.Close()

		decoder := json.NewDecoder(resp.Body)

		decoder.Decode(&response)

		for k, v := range response.Config.Banners {
			gtype := COLLABLCGACHA

			if k < STELLARBOUNDARY {
				gtype = STELLARGACHA
			} else if k < CHARACTERBOUNDARY {
				gtype = CHARACTERGACHA
			} else if k < LCBOUNDARY {
				gtype = LCGACHA
			} else if k < DEPARTUREBOUNDARY {
				gtype = DEPARTUREGACHA
			} else if k < COLLABCHARBOUNDARY {
				gtype = COLLABCHARACTERGACHA
			}

			banner := Banner{
				GachaType:       strconv.Itoa(gtype),
				Id:              strconv.Itoa(k),
				RateUp4:         v.RateUp4,
				RateUp:          v.RateUp,
				Rerun: v.Rerun,
				EndTime:         v.EndTime,
				StartTime:       v.StartTime,
				CompanionBanner: v.CompanionBanner,
				IsExclusive:     v.IsExclusive,
			}

			bannerIdx := slices.IndexFunc(index.Banners, func(b Banner) bool {
				return b.Id == banner.Id
			})

			if bannerIdx > -1 {
				banner.Desc = index.Banners[bannerIdx].Desc
				banner.Version = index.Banners[bannerIdx].Version
			} else {
				index.Banners = append(index.Banners, banner)
			}
		}

		// Sort ascending order by id first
		slices.SortStableFunc(index.Banners, func(i, j Banner) int {
			ival, _ := strconv.Atoi(i.Id)
			jval, _ := strconv.Atoi(j.Id)

			return ival - jval
		})

		// and then by gacha type so it becomes stellar > departure > event char > event lc > collab char > collab lc
		slices.SortStableFunc(index.Banners, func(i, j Banner) int {
			ival, _ := strconv.Atoi(i.GachaType)
			jval, _ := strconv.Atoi(j.GachaType)

			return ival - jval
		})
	})

	wg.Go(func() {
		lcResp, err := http.Get(LCURL)

		if err != nil {
			log.Fatalln("Unable to request light_cones.json from StarRailRes repository")
			return
		}

		defer lcResp.Body.Close()

		var lcResponse map[string]LightCone

		content, err := io.ReadAll(lcResp.Body)

		if err != nil {
			log.Fatalln("Unable to read light_cones.json content")
			return
		}

		json.Unmarshal(content, &lcResponse)

		lcIds := make(map[int]int, 0)

		for _, banner := range index.Banners {
			id, _ := strconv.Atoi(banner.Id)

			if id > 3000 && id < 4000 {
				bannerIdx := slices.IndexFunc(index.Banners, func(cmpBanner Banner) bool {
					return cmpBanner.Id == banner.Id
				})

				append := ""

				if !slices.Contains(slices.Collect(maps.Keys(lcIds)), banner.RateUp) {
					lcIds[banner.RateUp] = 0
				} else {
					lcIds[banner.RateUp] = lcIds[banner.RateUp] + 1
					banner.Rerun = lcIds[banner.RateUp]
				}

				if lcIds[banner.RateUp] > 0 {
					append = "Coalesced Truths: "
				}

				lcId := strconv.Itoa(banner.RateUp)

				banner.Desc = append + lcResponse[lcId].Name

				index.Banners[bannerIdx] = banner
			}
		}
	})

	wg.Wait()

	content, err := json.MarshalIndent(index, "", "    ")
	miniContent, err := json.Marshal(index)

	if err != nil {
		log.Fatalln("Unable to marshal content")
		return
	}

	err = writeToIndex(content, miniContent)

	if err != nil {
		log.Fatalln("Unable to write to index.json")
		return
	}
}

// Writes content to index.json and index.min.json
func writeToIndex(content []byte, miniContent []byte) error {
	cwd, err := os.Getwd()

	if err != nil {
		return err
	}

	file, err := os.OpenFile(cwd+"/"+INDEXFILE, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0666)
	miniFile, err := os.OpenFile(cwd+"/"+INDEXMINFILE, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0666)

	if err != nil {
		return err
	}

	defer file.Close()

	_, err = file.Write(content)
	_, err = miniFile.Write(miniContent)

	if err != nil {
		return err
	}

	return nil
}

// Returns the structured json content from index.json
func readIndex() (BannerHistory, error) {
	var index BannerHistory

	cwd, err := os.Getwd()

	if err != nil {
		return index, err
	}

	content, err := os.ReadFile(cwd + "/" + INDEXFILE)

	if err != nil {
		return index, err
	}

	err = json.Unmarshal(content, &index)

	if err != nil {
		return index, err
	}

	return index, nil
}

// Sets the version of the light cones appropriately with their respective character banners.
// For index generation purposes only.
func applyVersionToLc(index *BannerHistory) {
	charIndex := slices.Clone(index.Banners)
	lcIndex := slices.Clone(index.Banners)

	charIndex = slices.DeleteFunc(charIndex, func(b Banner) bool {
		id, _ := strconv.Atoi(b.Id)
		return id > CHARACTERBOUNDARY-1
	})

	lcIndex = slices.DeleteFunc(lcIndex, func(b Banner) bool {
		id, _ := strconv.Atoi(b.Id)
		return id < CHARACTERBOUNDARY || id > LCBOUNDARY-1
	})

	for i, j := range charIndex {
		index.Banners[i+len(charIndex)+1].Version = j.Version
	}
}

func getLatestBanner(index *BannerHistory, gachaType int) Banner {
	charIndex := slices.Clone(index.Banners)

	charIndex = slices.DeleteFunc(charIndex, func(b Banner) bool {
		id, _ := strconv.Atoi(b.Id)

		cond := false
		switch gachaType {
		case CHARACTERGACHA:
			cond = id > CHARACTERBOUNDARY-1
		case LCGACHA:
			cond = id < CHARACTERBOUNDARY || id > LCBOUNDARY-1
		case COLLABCHARACTERGACHA:
			cond = id < DEPARTUREBOUNDARY || id > COLLABCHARBOUNDARY-1
		case COLLABLCGACHA:
			cond = id < COLLABCHARBOUNDARY
		}

		return cond
	})

	return charIndex[len(charIndex)-1]
}

// Adds banner as the latest banner of the banner GachaType
// Should be used when new banners come out and after index.json is initialized with setup()
func addBanner(index *BannerHistory, banner Banner) error {
	gachaType, _ := strconv.Atoi(banner.GachaType)
	id, _ := strconv.Atoi(banner.Id)

	latestBanner := getLatestBanner(index, gachaType)
	latestId, _ := strconv.Atoi(latestBanner.Id)

	if id > 0 && latestId >= id {
		return errors.New("Latest Banner ID is equal to or more recent than the input banner ID")
	}

	latestIndex := slices.IndexFunc(index.Banners, func(b Banner) bool {
		return b.Id == latestBanner.Id
	})

	if banner.Rerun > 0 {
		append := "Indelible Coterie: "

		if gachaType == LCGACHA {
				append = "Coalesced Truths: "
		}
		
		banner.Desc = append + banner.Desc
	}

	banner.Version = strconv.FormatFloat(VERSION, 'f', 1,32)
	
	index.Banners = slices.Insert(index.Banners, latestIndex + 1, banner)
	
	return nil
}

func main() {
	monitor()
}
