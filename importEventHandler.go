package main

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/dukfaar/itemBackend/item"
	"github.com/globalsign/mgo/bson"
)

type RCItemEventData struct {
	Name        string `json:"name"`
	NamespaceID string `json:"namespace"`
	//Add other vars here
}

func setModelFromRCEvent(itemModel *item.Model, data RCItemEventData) {
	itemModel.Name = data.Name
	itemModel.NamespaceID = bson.ObjectIdHex(data.NamespaceID)
	//Add other vars here
}

func createItemModelFromRCEvent(itemService item.Service, itemData RCItemEventData) error {
	var itemModel = item.Model{}
	setModelFromRCEvent(&itemModel, itemData)

	_, err := itemService.Create(&itemModel)

	if err != nil {
		fmt.Printf("Error(%v) saving new item: %v\n", err, itemModel)
		return err
	}

	return nil
}

func updateItemModelFromRCEvent(itemService item.Service, itemModel *item.Model, itemData RCItemEventData) error {
	if itemModel == nil {
		fmt.Println("itemModel is ni")
		return errors.New("itemModel is nil")
	}

	setModelFromRCEvent(itemModel, itemData)

	_, err := itemService.Update(itemModel.ID.Hex(), itemModel)

	if err != nil {
		fmt.Printf("Error(%v) updating item: %v\n", err, itemModel)
		return err
	}

	return nil
}

func CreateRCEventImporter(itemService item.Service) func(msg []byte) error {
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
				return createItemModelFromRCEvent(itemService, itemData)
			} else {
				fmt.Printf("Unknown error: %v\n", err)
				return err
			}
		}

		return updateItemModelFromRCEvent(itemService, itemModel, itemData)
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
