package main

import (
	"encoding/json"
	"errors"
	"fmt"

	dukgraphql "github.com/dukfaar/goUtils/graphql"
	"github.com/dukfaar/itemBackend/item"
	"github.com/globalsign/mgo/bson"
)

type RCItemEventData struct {
	Name              string `json:"name" model:"Name"`
	NamespaceID       string `json:"namespace"`
	GatheringLevel    int32  `json:"gatheringLevel"`
	GatheringJob      string `json:"gatheringJob"`
	GatheringEffort   int32  `json:"gatheringEffort"`
	Price             int32  `json:"price"`
	PriceHQ           int32  `json:"priceHQ"`
	UnspoiledNode     bool   `json:"unspoiledNode"`
	UnspoiledNodeTime struct {
		Time           int32  `json:"time"`
		Duration       int32  `json:"duration"`
		AmPm           string `json:"ampm"`
		FolkloreNeeded string `json:"folkloreNeeded"`
	} `json:"unspoiledNodeTime"`
	AvailableFromNpc bool `json:"availableFromNpc"`
	//Add other vars here
}

var xivJobMap = make(map[string]string)

func fetchFFXIVJob(fetcher dukgraphql.Fetcher, jobName string, namespace string) (string, error) {
	if _, ok := xivJobMap[jobName]; ok {
		return xivJobMap[jobName], nil
	}

	fmt.Printf("fetching class %s in namespace %s\n", jobName, namespace)
	classResult, err := fetcher.Fetch(dukgraphql.Request{
		Query: "query { classByNameOrSynonym(name: \"" + jobName + "\", namespaceId: \"" + namespace + "\") { _id name } }",
	})

	if err != nil {
		fmt.Printf("Error fetching job: %v\n", err)
		return "", err
	}

	classResponse := dukgraphql.Response{classResult}
	fmt.Printf("result: %+v\n", classResponse)

	result := classResponse.GetObject("classByNameOrSynonym").GetString("_id")
	xivJobMap[jobName] = result
	return result, nil
}

func setModelFromRCEvent(itemModel *item.Model, data RCItemEventData, fetcher dukgraphql.Fetcher) {
	itemModel.Name = data.Name
	itemModel.NamespaceID = bson.ObjectIdHex(data.NamespaceID)
	itemModel.GatheringEffort = &data.GatheringEffort

	gatheringJobChannel := make(chan *bson.ObjectId)
	go func() {
		job, err := fetchFFXIVJob(fetcher, data.GatheringJob, data.NamespaceID)
		if err != nil {
			gatheringJobChannel <- nil
			return
		}
		gatheringJobId := bson.ObjectIdHex(job)
		gatheringJobChannel <- &gatheringJobId
	}()

	itemModel.GatheringLevel = &data.GatheringLevel
	itemModel.Price = &data.Price
	itemModel.PriceHQ = &data.PriceHQ
	itemModel.UnspoiledNode = &data.UnspoiledNode
	itemModel.UnspoiledNodeTime = &item.UnspoiledNodeTime{
		Time:           &data.UnspoiledNodeTime.Time,
		Duration:       &data.UnspoiledNodeTime.Duration,
		AmPm:           &data.UnspoiledNodeTime.AmPm,
		FolkloreNeeded: &data.UnspoiledNodeTime.FolkloreNeeded,
	}
	itemModel.AvailableFromNpc = &data.AvailableFromNpc
	//Add other vars here

	//add delayed fetches here
	itemModel.GatheringJobID = <-gatheringJobChannel
}

func createItemModelFromRCEvent(itemService item.Service, itemData RCItemEventData, fetcher dukgraphql.Fetcher) error {
	var itemModel = item.Model{}
	setModelFromRCEvent(&itemModel, itemData, fetcher)

	_, err := itemService.Create(&itemModel)

	if err != nil {
		fmt.Printf("Error(%v) saving new item: %v\n", err, itemModel)
		return err
	}

	return nil
}

func updateItemModelFromRCEvent(itemService item.Service, itemModel *item.Model, itemData RCItemEventData, fetcher dukgraphql.Fetcher) error {
	if itemModel == nil {
		fmt.Println("itemModel is ni")
		return errors.New("itemModel is nil")
	}

	setModelFromRCEvent(itemModel, itemData, fetcher)

	_, err := itemService.Update(itemModel.ID.Hex(), itemModel)

	if err != nil {
		fmt.Printf("Error(%v) updating item: %v\n", err, itemModel)
		return err
	}

	return nil
}

func CreateRCEventImporter(itemService item.Service, fetcher dukgraphql.Fetcher) func(msg []byte) error {
	return func(msg []byte) error {
		var itemData RCItemEventData
		err := json.Unmarshal(msg, &itemData)

		if err != nil {
			fmt.Printf("Error(%v) unmarshaling event data: %v\n", err, string(msg))
			return err
		}

		if itemData.Name == "" {
			fmt.Printf("Cant import an item without a name\n")
			return errors.New("Item has no Name")
		}

		itemModel, err := itemService.FindByName(itemData.Name)

		if err != nil {
			if err.Error() == "not found" {
				return createItemModelFromRCEvent(itemService, itemData, fetcher)
			} else {
				fmt.Printf("Unknown error: %v\n", err)
				return err
			}
		}

		return updateItemModelFromRCEvent(itemService, itemModel, itemData, fetcher)
	}
}

type XivdbItemEventData struct {
	ID          int32  `json:"id"`
	NameEN      string `json:"name_en"`
	NamespaceID string `json:"namespace"`
	//Add other vars here
}

func setModelFromXivdbEvent(itemModel *item.Model, data XivdbItemEventData) {
	itemModel.Name = data.NameEN
	itemModel.NamespaceID = bson.ObjectIdHex(data.NamespaceID)

	if itemModel.XivdbID == nil {
		itemModel.XivdbID = new(int32)
	}
	*itemModel.XivdbID = data.ID

	//Add other vars here
}

func createItemModelFromXivdbEvent(itemService item.Service, itemData XivdbItemEventData) error {
	var itemModel = item.Model{}
	setModelFromXivdbEvent(&itemModel, itemData)

	_, err := itemService.Create(&itemModel)

	if err != nil {
		fmt.Printf("Error(%v) creating item: %v\n", err, itemModel)
		return err
	}

	return nil
}

func updateItemModelFromXivdbEvent(itemService item.Service, itemModel *item.Model, itemData XivdbItemEventData) error {
	if itemModel == nil {
		fmt.Println("itemModel is ni")
		return errors.New("itemModel is nil")
	}

	setModelFromXivdbEvent(itemModel, itemData)

	_, err := itemService.Update(itemModel.ID.Hex(), itemModel)

	if err != nil {
		fmt.Printf("Error(%v) updating item: %v\n", err, itemModel)
		return err
	}

	return nil
}

func CreateXivdbEventImporter(itemService item.Service) func(msg []byte) error {
	return func(msg []byte) error {
		var itemData XivdbItemEventData
		err := json.Unmarshal(msg, &itemData)

		if err != nil {
			fmt.Printf("Error(%v) unmarshaling event data: %v\n", err, string(msg))
			return err
		}

		itemModel, err := itemService.FindByXivdbID(itemData.ID)

		if err != nil {
			if err.Error() == "not found" {
				itemModel, err := itemService.FindByName(itemData.NameEN)

				if err != nil {
					if err.Error() == "not found" {
						return createItemModelFromXivdbEvent(itemService, itemData)
					}
				} else {
					return updateItemModelFromXivdbEvent(itemService, itemModel, itemData)
				}
			} else {
				return err
			}
		}

		return updateItemModelFromXivdbEvent(itemService, itemModel, itemData)
	}
}
