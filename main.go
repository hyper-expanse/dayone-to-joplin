package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

type Center struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type Region struct {
	Center Center `json:"center"`
}

type Location struct {
	Address            string  `json:"address"`
	AdministrativeArea string  `json:"administrativeArea,omitempty"`
	Country            string  `json:"country,omitempty"`
	Latitude           float64 `json:"latitude"`
	LocalityName       string  `json:"localityName,omitempty"`
	Longitude          float64 `json:"longitude"`
	PlaceName          string  `json:"placeName"`
	Region             Region  `json:"region"`
}

type Weather struct {
	ConditionalDescription string  `json:"conditionsDescription"`
	PressureMB             float64 `json:"pressureMB"`
	RelativeHumidity       float64 `json:"relativeHumidity"`
	TemperatureCelsius     float64 `json:"temperatureCelsius"`
	VisibilityKM           float64 `json:"visibilityKM"`
	WeatherCode            string  `json:"weatherCode"`
	WeatherServiceName     string  `json:"weatherServiceName"`
	WindBearing            float64 `json:"windBearing"`
	WindChillCelsius       float64 `json:"windChillCelsius"`
	WindSpeedKPH           float64 `json:"windSpeedKPH"`
}

type JournalEntries struct {
	Entries  []JournalEntry `json:"entries"`
	Metadata struct {
		Version string `json:"version"`
	} `json:"metadata"`
}

type Photo struct {
	ExposureBiasValue float64 `json:"exposureBiasValue"`
	Height            float64 `json:"height"`
	Identifier        string  `json:"identifier"`
	MD5               string  `json:"md5"`
	OrderInEntry      float64 `json:"orderInEntry"`
	Type              string  `json:"type"`
	Width             float64 `json:"width"`
}

type JournalEntry struct {
	Audios       []string  `json:"audios"`
	CreationDate time.Time `json:"creationDate"`
	Location     *Location `json:"location,omitempty"`
	Photos       []Photo   `json:"photos"`
	Starred      bool      `json:"starred"`
	Tags         []string  `json:"tags"`
	Text         string    `json:"text"`
	TimeZone     string    `json:"timeZone"`
	UUID         string    `json:"uuid"`
	Weather      *Weather  `json:"weather,omitempty"`
}

type TagItem struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type Tags struct {
	Items   []TagItem `json:"items"`
	HasMore bool      `json:"has_more"`
}

type Note struct {
	ParentID        string  `json:"parent_id"`
	Title           string  `json:"title"`
	Body            string  `json:"body"`
	UserCreatedTime int64   `json:"user_created_time"`
	UserUpdatedTime int64   `json:"user_updated_time"`
	Latitude        float64 `json:"latitude"`
	Longitude       float64 `json:"longitude"`
}

type NoteResponse struct {
	ID string `json:"id"`
}

type Tag struct {
	Title string `json:"title"`
}

type TagResponse struct {
	ID string `json:"id"`
}

type ResourceResponse struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	Mime          string `json:"mime"`
	Filename      string `json:"filename"`
	FileExtension string `json:"file_extension"`
	Size          int64  `json:"size"`
}

func main() {
	var host string
	var journalFolder string
	var notebook string
	var token string

	flag.StringVar(&host, "host", "http://localhost:41184", "Fully qualified host address of your local Joplin instance.")
	flag.StringVar(&journalFolder, "journalFolder", "", "Full path to the directory containing your extracted journal data.")
	flag.StringVar(&notebook, "notebook", "44538ac414c340af8eba12fef4066446", "ID of the notebook to import your journal entries into.")
	flag.StringVar(&token, "token", "", "API token for your local Joplin instance.")
	flag.Parse()

	client := &http.Client{
		Timeout: time.Second * 30,
	}

	rawData, err := ioutil.ReadFile(fmt.Sprintf("%s/AllEntries.json", journalFolder))
	if err != nil {
		panic(err)
	}

	var parsedData JournalEntries
	err = json.Unmarshal(rawData, &parsedData)
	if err != nil {
		panic(err)
	}

	tags, err := GetTags(host, token)
	if err != nil {
		panic(err)
	}

	for _, journalEntry := range parsedData.Entries {
		title := strings.Replace(strings.Split(journalEntry.Text, "\n")[0], "# ", "", -1)
		fmt.Printf("Journal Entry: %s\n", title)

		text := strings.Join(strings.Split(journalEntry.Text, "\n")[1:], "\n")

		for _, photo := range journalEntry.Photos {
			resource, err := CreateResource(journalFolder, photo, host, token)
			if err != nil {
				panic(err)
			}

			re := regexp.MustCompile(`\!\[\]\(dayone-moment://` + photo.Identifier + `\)`)
			text = re.ReplaceAllString(text, fmt.Sprintf("![](:/%s)", resource.ID))
		}

		// If a journal entry contains no location we set a default location so we can reference it later.
		if journalEntry.Location == nil {
			journalEntry.Location = &Location{
				Region: Region{
					Center: Center{
						Latitude:  0.0,
						Longitude: 0.0,
					},
				},
			}
		}

		// Prepend the date to help with ordering of journal entries imported into a Joplin notebook
		// containing other entires prepended with the date (Because not all entries are written on the
		// date they are about.
		date := journalEntry.CreationDate
		title = fmt.Sprintf("%d-%02d-%02d %s", date.Year(), date.Month(), date.Day(), title)

		note := Note{
			notebook,
			title,
			text,
			// Apparently you have to multiply the number by a thousand
			// - https://github.com/laurent22/joplin/issues/5224#issuecomment-886241875
			journalEntry.CreationDate.Unix() * 1000,
			journalEntry.CreationDate.Unix() * 1000,
			journalEntry.Location.Region.Center.Latitude,
			journalEntry.Location.Region.Center.Longitude,
		}

		noteText, err := json.Marshal(note)
		if err != nil {
			panic(err)
		}

		url := fmt.Sprintf("%s/notes?token=%s", host, token)

		request, err := http.NewRequest("POST", url, bytes.NewBuffer(noteText))
		if err != nil {
			panic(err)
		}

		response, err := client.Do(request)
		if err != nil {
			panic(err)
		}
		defer response.Body.Close()

		noteResponse, err := ioutil.ReadAll(response.Body)
		if err != nil {
			panic(err)
		}

		noteEntryResponse := NoteResponse{}
		err = json.Unmarshal(noteResponse, &noteEntryResponse)
		if err != nil {
			panic(err)
		}

		for _, tag := range journalEntry.Tags {
			tag = strings.ToLower(tag)
			fmt.Printf("  tag: %s\n", tag)

			tagID := ""
			for _, existingTag := range tags.Items {
				if existingTag.Title == tag {
					fmt.Printf("  found match - %s == %s\n", existingTag.Title, tag)
					tagID = existingTag.ID
					break
				}
			}

			fmt.Printf("  tag ID: %s\n", tagID)
			if len(tagID) == 0 {
				tag := Tag{
					tag,
				}

				tagText, err := json.Marshal(tag)
				if err != nil {
					panic(err)
				}

				url := fmt.Sprintf("%s/tags?token=%s", host, token)

				request, err := http.NewRequest("POST", url, bytes.NewBuffer(tagText))
				if err != nil {
					panic(err)
				}

				response, err := client.Do(request)
				if err != nil {
					panic(err)
				}
				defer response.Body.Close()

				if response.StatusCode != http.StatusOK {
					fmt.Printf("Post error status: %s\n", response.Status)
					body, err := ioutil.ReadAll(response.Body)
					if err != nil {
						panic(err)
					}
					panic(errors.New(string(body)))
				}

				body, err := ioutil.ReadAll(response.Body)
				if err != nil {
					panic(err)
				}

				tagResponse := TagResponse{}
				err = json.Unmarshal(body, &tagResponse)
				if err != nil {
					panic(err)
				}

				tagID = tagResponse.ID
				fmt.Printf("  created ID: %s\n", tagID)
			}

			url = fmt.Sprintf("%s/tags/%s/notes?token=%s", host, tagID, token)

			request, err = http.NewRequest("POST", url, bytes.NewBuffer(noteResponse))
			if err != nil {
				panic(err)
			}

			response, err = client.Do(request)
			if err != nil {
				panic(err)
			}
			defer response.Body.Close()
		}
	}
}

func GetTags(host, token string) (Tags, error) {
	var tags Tags

	url := fmt.Sprintf("%s/tags?token=%s", host, token)
	client := &http.Client{
		Timeout: time.Second * 30,
	}

	page := 0
	for {
		var tagPage Tags

		page = page + 1
		pageUrl := fmt.Sprintf("%s&page=%d", url, page)
		request, err := http.NewRequest("GET", pageUrl, nil)
		if err != nil {
			return tags, err
		}

		response, err := client.Do(request)
		if err != nil {
			return tags, err
		}
		defer response.Body.Close()

		tagsResponse, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return tags, err
		}

		err = json.Unmarshal(tagsResponse, &tagPage)
		if err != nil {
			return tags, err
		}

		tags.Items = append(tags.Items, tagPage.Items...)

		if !tagPage.HasMore {
			break
		}
	}

	return tags, nil
}

func CreateResource(journalFolder string, photo Photo, host string, token string) (ResourceResponse, error) {
	var resourceResponse ResourceResponse

	client := &http.Client{
		Timeout: time.Second * 30,
	}

	photoPath := fmt.Sprintf("%s/photos/%s.%s", journalFolder, photo.MD5, photo.Type)

	var buffer bytes.Buffer
	multipartWriter := multipart.NewWriter(&buffer)

	file, err := os.Open(photoPath)
	if err != nil {
		return resourceResponse, err
	}
	defer file.Close()
	fileWriter, err := multipartWriter.CreateFormFile("data", file.Name())
	if err != nil {
		return resourceResponse, err
	}
	_, err = io.Copy(fileWriter, file)
	if err != nil {
		return resourceResponse, err
	}

	propsWriter, err := multipartWriter.CreateFormField("props")
	if err != nil {
		return resourceResponse, err
	}
	_, err = io.Copy(propsWriter, strings.NewReader("{}"))
	if err != nil {
		return resourceResponse, err
	}

	multipartWriter.Close()

	url := fmt.Sprintf("%s/resources?token=%s", host, token)
	request, err := http.NewRequest("POST", url, &buffer)
	if err != nil {
		return resourceResponse, err
	}

	request.Header.Set("Content-Type", multipartWriter.FormDataContentType())

	response, err := client.Do(request)
	if err != nil {
		return resourceResponse, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		fmt.Printf("Post error status: %s", response.Status)
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return resourceResponse, err
		}
		return resourceResponse, errors.New(string(body))
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return resourceResponse, err
	}

	err = json.Unmarshal(body, &resourceResponse)
	if err != nil {
		return resourceResponse, err
	}

	return resourceResponse, nil
}
